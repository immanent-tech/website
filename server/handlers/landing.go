// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package handlers

import "net/http"

type LandingPage struct{}

func NewLandingPage() http.HandlerFunc {
	page := &LandingPage{}
	return page.ServeHTTP
}

func (h *LandingPage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.Get(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *LandingPage) Get(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusNotImplemented)
}
