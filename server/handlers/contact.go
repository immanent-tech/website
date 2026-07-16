// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/a-h/templ"
	"github.com/immanent-tech/go-base/validation"
	"github.com/immanent-tech/www-immanent-tech/providers/fastmail"
	"github.com/immanent-tech/www-immanent-tech/server/forms"
	"github.com/immanent-tech/www-immanent-tech/web/templates"
	slogctx "github.com/veqryn/slog-context"
)

type ContactPage struct {
	template templ.Component
}

func (p *ContactPage) FullResponse(w http.ResponseWriter, r *http.Request) {
	templ.Handler(p.template).ServeHTTP(w, r)
}

func (p *ContactPage) PartialResponse(w http.ResponseWriter, r *http.Request) {
	templ.Handler(p.template, templ.WithFragments(templates.BodyFragment)).ServeHTTP(w, r)
}

func Contact() http.HandlerFunc {
	page := &ContactPage{
		template: templates.Page(templates.Contact()),
	}
	return RenderPage(page)
}

type ContactRequest struct {
	// ContactEmail is the email address the entered for getting in touch about the issue.
	ContactEmail string `form:"contact_email" json:"contact_email" validate:"required,email"`

	// Details is the text about the issue.
	Details string `form:"details" json:"details" validate:"required"`
}

func (r *ContactRequest) Valid() error {
	if err := validation.Validate.Struct(r); err != nil {
		return fmt.Errorf("contact request invalid: %w", err)
	}
	return nil
}

func (r *ContactRequest) Sanitise() error {
	r.ContactEmail = validation.SanitizeString(r.ContactEmail)
	r.Details = validation.SanitizeString(r.Details)
	return nil
}

func HandleSubmitContact() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// Validate the subscription issue request.
		request, valid, err := forms.DecodeMultiPartForm[*ContactRequest](req)
		if err != nil || !valid {
			slogctx.FromCtx(req.Context()).Error("Could not decode contact form submission.",
				slog.Any("error", err),
			)
			http.Error(res, "Invalid Params", http.StatusUnprocessableEntity)
			return
		}

		sender, err := mail.ParseAddress(request.ContactEmail)
		if err != nil {
			slogctx.FromCtx(req.Context()).Error("Could not parse email address.",
				slog.Any("error", err),
			)
			http.Error(res, "Invalid Params", http.StatusUnprocessableEntity)
			return
		}

		// Build issue body.
		var bodyBuilder strings.Builder
		bodyBuilder.WriteString("Contact Email: ")
		bodyBuilder.WriteString(request.ContactEmail)
		bodyBuilder.WriteRune('\n')
		bodyBuilder.WriteString("Details:")
		bodyBuilder.WriteRune('\n')
		bodyBuilder.WriteString(request.Details)
		bodyBuilder.WriteRune('\n')

		if err := fastmail.SendEmail(sender, "Contact Form Submission", bodyBuilder.String()); err != nil {
			slogctx.FromCtx(req.Context()).Error("Could not send email.",
				slog.Any("error", err),
			)
			http.Error(res, "Request Failed", http.StatusInternalServerError)
			return
		}

		// Show notification of issue reported.
		RenderPartial(&Notification{
			notification: &templates.Notification{
				Title: "Thanks for contacting us!",
				Description: new(
					"If we need to reach out to discuss, we will send you an email to the address that was submitted.",
				),
				Status: http.StatusOK,
			},
		}).ServeHTTP(res, req)
	}

}
