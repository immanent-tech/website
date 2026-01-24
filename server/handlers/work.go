// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/immanent-tech/www-immanent-tech/web/templates"
)

type WorkPage struct {
	template templ.Component
}

func NewWorkPage() http.HandlerFunc {
	page := &WorkPage{
		template: templates.Page(templates.Work()),
	}
	return RenderPage(page)
}

func (p *WorkPage) FullResponse(w http.ResponseWriter, r *http.Request) {
	templ.Handler(p.template).ServeHTTP(w, r)
}

func (p *WorkPage) PartialResponse(w http.ResponseWriter, r *http.Request) {
	templ.Handler(p.template, templ.WithFragments(templates.BodyFragment)).ServeHTTP(w, r)
}
