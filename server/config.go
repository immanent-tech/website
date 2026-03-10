// Copyright 2024 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package server

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/immanent-tech/www-immanent-tech/config"
	"github.com/immanent-tech/www-immanent-tech/validation"
)

const (
	serverConfigEnvPrefix   = config.EnvPrefix
	defaultCompressionLevel = 5
)

var compressMimetypes = []string{"text/html", "text/css", "text/javascript", "font/woff2", "image/svg+xml"}

var cfg Config

type timeout string

func (t timeout) Validate() error {
	if _, err := time.ParseDuration(string(t)); err != nil {
		return fmt.Errorf("parse timeout: %w", err)
	}
	return nil
}

func (t timeout) Duration() time.Duration {
	duration, err := time.ParseDuration(string(t))
	if err != nil {
		return time.Minute
	}
	return duration
}

// Config contains the server configuration options.
type Config struct {
	Port         uint64  `koanf:"port"         validate:"port"`
	Host         string  `koanf:"host"         validate:"hostname|fqdn|ip"`
	CertFile     string  `koanf:"crt"          validate:"omitempty,file"`
	KeyFile      string  `koanf:"key"          validate:"omitempty,file"`
	ReadTimeout  timeout `koanf:"readtimeout"  validate:"required,validateFn"`
	WriteTimeout timeout `koanf:"writetimeout" validate:"required,validateFn"`
	IdleTimeout  timeout `koanf:"idletimeout"  validate:"required,validateFn"`
}

// loadConfigOnce loads the server configuration and ensures this is only done
// one time, no matter how many times it is called.
var loadConfigOnce = sync.OnceValue(func() error {
	// Load server config.
	if err := config.Load(serverConfigEnvPrefix, &cfg); err != nil {
		return fmt.Errorf("load server environment: %w", err)
	}
	// Load additional environment variables.
	if os.Getenv("PORT") != "" {
		if port, err := strconv.ParseUint(os.Getenv("PORT"), 10, 64); err != nil {
			return fmt.Errorf("load port: %w", err)
		} else {
			cfg.Port = port
		}
	}

	if err := validation.Validate.Struct(cfg); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	return nil
})
