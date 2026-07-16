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
	slogctx "github.com/veqryn/slog-context"

	"github.com/immanent-tech/www-immanent-tech/server/handlers"
	"github.com/immanent-tech/www-immanent-tech/server/middlewares"
	"github.com/immanent-tech/www-immanent-tech/web"

	"github.com/immanent-tech/go-base/server/middlewares/etag"
	"github.com/immanent-tech/go-base/server/middlewares/security"
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

	// Set up routes.
	// rateLimiter := middlewares.NewRateLimiter()

	// Set up a new chi router.
	router := chi.NewRouter()

	// Health check endpoints (for GCP).
	router.Use(middleware.Heartbeat("/health-check"))

	// Standard middleware stack.
	router.Use(
		middleware.RequestID,
		middlewares.Logger,
		middleware.Recoverer,
		security.SetupCORS,
		security.ContentSecurityPolicy,
		security.GeneralSecurity,
		security.CrossOriginProtection,
		security.GeneralSecurity,
		security.PreventCSRF,
		middleware.Compress(defaultCompressionLevel, compressMimetypes...),
		middleware.StripSlashes,
		etag.Etag,
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
			etag.Etag,
		)
		r.Get("/", handlers.NewLandingPage())
		r.Get("/work", handlers.NewWorkPage())
		r.Get("/contact", handlers.Contact())
		r.Post("/contact", handlers.HandleSubmitContact())
	})

	svr := &http.Server{
		Protocols:         new(http.Protocols),
		Handler:           router,
		Addr:              net.JoinHostPort(cfg.Host, strconv.FormatUint(cfg.Port, 10)),
		ReadHeaderTimeout: cfg.ReadTimeout.Duration,
		ReadTimeout:       cfg.ReadTimeout.Duration,
		WriteTimeout:      cfg.WriteTimeout.Duration,
		IdleTimeout:       cfg.IdleTimeout.Duration,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
	svr.Protocols.SetUnencryptedHTTP2(true) // Enable H2C (HTTP/2 cleartext)
	svr.Protocols.SetHTTP1(true)            // Enable HTTP/1.1
	svr.Protocols.SetHTTP2(false)           // Explicitly disable encrypted HTTP/2 (HTTPS)

	logger.Info("Starting server...",
		slog.String("address", svr.Addr),
		slog.Duration("read_timeout", cfg.ReadTimeout.Duration),
		slog.Duration("write_timeout", cfg.WriteTimeout.Duration),
		slog.Duration("idle_timeout", cfg.IdleTimeout.Duration),
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
	if err := svr.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server failed to shutdown gracefully.",
			slog.Any("error", err),
			slog.Time("stop_time", time.Now()),
		)
	}

	logger.Info("Server shutdown gracefully",
		slog.Time("stop_time", time.Now()),
	)

	return nil
}
