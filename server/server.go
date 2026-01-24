// Copyright 2024 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/justinas/nosurf"
	slogctx "github.com/veqryn/slog-context"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/immanent-tech/www-immanent-tech/server/handlers"
	"github.com/immanent-tech/www-immanent-tech/server/middlewares"
	"github.com/immanent-tech/www-immanent-tech/web"
)

const (
	gracefulShutdownTimeout = 30 * time.Second
)

// Start will start the server.
func Start(logger *slog.Logger) error {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelFunc()

	ctx = slogctx.NewCtx(ctx, logger)

	// Load the server config.
	if err := loadConfigOnce(); err != nil {
		return fmt.Errorf("unable to load server config: %w", err)
	}

	var err error

	// Set up routes.
	rateLimiter := middlewares.NewRateLimiter()

	// Set up a new chi router.
	router := chi.NewRouter()

	// Health check endpoints (for GCP).
	router.Use(middleware.Heartbeat("/health-check"))

	// Standard middleware stack.
	router.Use(
		middleware.RequestID,
		middlewares.Logger,
		middleware.Recoverer,
		middlewares.SetupCORS,
		middlewares.CrossOriginProtection,
		middlewares.ContentSecurityPolicy,
		middlewares.GeneralSecurity,
		middlewares.SaveCSRFToken,
		middlewares.RateLimit(rateLimiter),
		middleware.Compress(defaultCompressionLevel, compressMimetypes...),
		middleware.StripSlashes,
		middlewares.Etag,
		middlewares.CrossOriginProtection,
		middlewares.SetupHTMX,
	)

	// Error handling.
	router.NotFound(handlers.NotFound())
	// Static content.
	router.Handle("/content/*", handlers.StaticFileHandler(http.FS(web.StaticContentFS)))
	router.Handle("/robots.txt", handlers.RobotsHandler())

	// Public facing routes.
	router.Group(func(r chi.Router) {
		r.Use(
			middlewares.Etag,
		)
		r.Get("/", handlers.NewLandingPage())
		// r.Get("/about", handlers.About())
	})

	// Authenticated routes.
	// router.Group(func(r chi.Router) {
	// 	r.Use(
	// 		middlewares.Etag,
	// 		middlewares.CrossOriginProtection,
	// 		middlewares.SetupHTMX,
	// 		session.LoadAndSave,
	// 		middlewares.RequireUserAuth,
	// 		middlewares.RefreshTokenIfNeeded,
	// 		middlewares.SetCacheControl,
	// 		// middleware.NoCache,
	// 	)
	// })

	csrfRouter := nosurf.New(router)
	csrfRouter.SetFailureHandler(middlewares.CSRFError())

	h2s := &http2.Server{}
	svr := &http.Server{
		Handler:      h2c.NewHandler(csrfRouter, h2s),
		Addr:         net.JoinHostPort(cfg.Host, strconv.FormatUint(cfg.Port, 10)),
		ReadTimeout:  cfg.ReadTimeout.Duration(),
		WriteTimeout: cfg.WriteTimeout.Duration(),
		IdleTimeout:  cfg.IdleTimeout.Duration(),
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	err = http2.ConfigureServer(svr, h2s)
	if err != nil {
		return fmt.Errorf("unable to configure server for H2C: %w", err)
	}

	logger.Info("Starting server...",
		slog.String("address", svr.Addr),
		slog.Time("start_time", time.Now()),
	)

	// And we serve HTTP until the world ends.
	go func() {
		var err error
		if cfg.CertFile != "" && cfg.KeyFile != "" {
			logger.Info("Using https.",
				slog.String("certificate file", cfg.CertFile),
				slog.String("key file", cfg.KeyFile),
			)
			err = svr.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
		} else {
			logger.Info("Using http.")
			err = svr.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Error("Could not listen.",
				slog.Any("error", err),
			)
		}
	}()

	<-ctx.Done()

	// Create shutdown context with 30-second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	// Trigger graceful shutdown
	logger.Info("Shutting down server...")
	if err := svr.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server failed to shutdown gracefully.",
			slog.Any("error", err),
		)
	}

	logger.Info("Server shutdown gracefully")

	return nil
}
