// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/immanent-tech/www-immanent-tech/web/templates"
)

type LandingPage struct {
	template templ.Component
}

func NewLandingPage() http.HandlerFunc {
	page := &LandingPage{
		template: templates.Page(templates.Landing()),
	}
	return RenderPage(page)
}

func (p *LandingPage) FullResponse(w http.ResponseWriter, r *http.Request) {
	templ.Handler(p.template).ServeHTTP(w, r)
}

func (p *LandingPage) PartialResponse(w http.ResponseWriter, r *http.Request) {
	templ.Handler(p.template, templ.WithFragments(templates.BodyFragment)).ServeHTTP(w, r)
}
