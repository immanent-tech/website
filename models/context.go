// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package models

import "context"

const (
	csrfTokenCtxKey contextKey = "csrfToken"
)

type contextKey string

// CSRFTokenToCtx stores the current valid CSRF token in the context.
func CSRFTokenToCtx(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, csrfTokenCtxKey, token)
}

// CSRFTokenFromCtx retrieves the current valid CSRF token from the context.
func CSRFTokenFromCtx(ctx context.Context) string {
	if token, ok := ctx.Value(csrfTokenCtxKey).(string); ok {
		return token
	}
	return ""
}
