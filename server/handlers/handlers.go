// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/immanent-tech/www-immanent-tech/web"
	slogctx "github.com/veqryn/slog-context"
)

// StaticFileHandler handles serving content from the embedded filesystem containing static assets (i.e., images,
// etc.).
func StaticFileHandler(fs http.FileSystem) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// Check, if the requested file is existing.
		if _, err := fs.Open(req.URL.Path); err != nil {
			// If file is not found, return HTTP 404 error.
			http.NotFound(res, req)
			return
		}
		switch {
		case strings.HasSuffix(req.URL.Path, "js"):
			// JS files are cached for 1 week.
			res.Header().Set("Cache-Control", "public, max-age=604800")
		case strings.HasSuffix(req.URL.Path, "css"):
			// CSS files are cached for 1 week.
			res.Header().Set("Cache-Control", "public, max-age=604800")
		case strings.HasSuffix(req.URL.Path, "woff2"):
			// Fonts are cached for 1 year.
			res.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case strings.HasSuffix(req.URL.Path, "png"):
			fallthrough
		case strings.HasSuffix(req.URL.Path, "jpg"):
			fallthrough
		case strings.HasSuffix(req.URL.Path, "webp"):
			fallthrough
		case strings.HasSuffix(req.URL.Path, "svg"):
			res.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		default:
			// Default is to cache for 1 week.
			res.Header().Set("Cache-Control", "public, max-age=604800, s-maxage=43200")
		}
		// File is found, return to standard http.FileServer.
		http.FileServer(fs).ServeHTTP(res, req)
	}
}

var robotsTxt []byte
var loadRobotsTxt = sync.OnceValue(func() error {
	var err error
	robotsTxt, err = web.StaticContentFS.ReadFile("content/robots.txt")
	if err != nil {
		return fmt.Errorf("read robots.txt: %w", err)
	}
	return nil
})

// RobotsHandler handles requests for robots.txt. In the future, it may handle more requests from non natural human
// clients...
func RobotsHandler() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if err := loadRobotsTxt(); err != nil {
			http.NotFound(res, req)
			return
		}
		res.Header().Set("Cache-Control", "public, max-age=604800, s-maxage=43200")
		res.WriteHeader(http.StatusOK)
		if _, err := res.Write(robotsTxt); err != nil {
			slogctx.FromCtx(req.Context()).Error("Unable to send robots.txt response.",
				slog.Any("error", err),
			)
		}
	}
}
