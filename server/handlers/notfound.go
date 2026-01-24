// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package handlers

import "net/http"

// NotFound handles showing a page for a 404 response.
func NotFound() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		http.NotFound(res, req)
	}
}
