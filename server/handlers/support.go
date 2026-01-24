// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/immanent-tech/www-immanent-tech/web/templates"
)

type SupportPage struct {
	template templ.Component
}

func Support() http.HandlerFunc {
	page := &SupportPage{
		template: templates.Page(templates.Support()),
	}
	return RenderPage(page)
}

func (p *SupportPage) FullResponse(w http.ResponseWriter, r *http.Request) {
	templ.Handler(p.template).ServeHTTP(w, r)
}

func (p *SupportPage) PartialResponse(w http.ResponseWriter, r *http.Request) {
	templ.Handler(p.template, templ.WithFragments(templates.BodyFragment)).ServeHTTP(w, r)
}
