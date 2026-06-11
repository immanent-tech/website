// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package fastmail

import (
	"fmt"
	"net/mail"
	"sync"

	"github.com/cwinters8/gomap"
	"github.com/immanent-tech/www-immanent-tech/config"
	"github.com/immanent-tech/www-immanent-tech/validation"
)

const (
	configPrefix = "FASTMAIL_"
	apiEndpoint  = "https://api.fastmail.com/jmap/session"
)

// Config contains the server configuration options.
type Config struct {
	APIKey   string `koanf:"apikey"   validate:"required"`
	Identity string `koanf:"identity" validate:"required,email"`
}

var cfg Config

// loadConfig loads the server configuration and ensures this is only done
// one time, no matter how many times it is called.
var loadConfig = sync.OnceValue(func() error {
	if err := config.Load(configPrefix, &cfg); err != nil {
		return fmt.Errorf("load config from environment: %w", err)
	}
	if err := validation.Validate.Struct(cfg); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	return nil
})

var client *gomap.Client

var loadClient = sync.OnceValue(func() error {
	if err := loadConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var err error
	client, err = gomap.NewClient(
		apiEndpoint,
		cfg.APIKey,
		gomap.DefaultDrafts,
		gomap.DefaultSent,
	)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}

	return nil
})

func SendEmail(from *mail.Address, subject, body string) error {
	if err := loadClient(); err != nil {
		return fmt.Errorf("load client: %w", err)
	}
	if err := client.SendEmailWithIdentity(
		gomap.NewAddresses(gomap.NewAddress(from.Name, from.Address)),
		gomap.NewAddresses(gomap.NewAddress("Immanent Tech", "hello@immanent.tech")),
		subject,
		body,
		cfg.Identity,
		false,
	); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}
