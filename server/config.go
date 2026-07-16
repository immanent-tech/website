// Copyright 2024 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/immanent-tech/go-base/config"
	"github.com/immanent-tech/go-base/validation"
)

const (
	configEnvPrefix         = "WWW_"
	defaultCompressionLevel = 5
)

var compressMimetypes = []string{"text/html", "text/css", "text/javascript", "font/woff2", "image/svg+xml"}

var cfg = Config{
	Host:         "0.0.0.0",
	ReadTimeout:  config.NewDuration(120 * time.Second),
	WriteTimeout: config.NewDuration(30 * time.Second),
	IdleTimeout:  config.NewDuration(900 * time.Second),
}

// Config contains the server configuration options.
type Config struct {
	Port         uint64          `koanf:"port"         validate:"required,port"`
	Host         string          `koanf:"host"         validate:"omitempty,hostname|fqdn|ip"`
	CertFile     string          `koanf:"crt"          validate:"omitempty,file"`
	KeyFile      string          `koanf:"key"          validate:"omitempty,file"`
	ReadTimeout  config.Duration `koanf:"readtimeout"  validate:"omitempty"`
	WriteTimeout config.Duration `koanf:"writetimeout" validate:"omitempty"`
	IdleTimeout  config.Duration `koanf:"idletimeout"  validate:"omitempty"`
}

// loadConfigOnce loads the server configuration and ensures this is only done
// one time, no matter how many times it is called.
var loadConfigOnce = sync.OnceValue(func() error {
	// Load server config.
	if err := config.Load(configEnvPrefix, &cfg); err != nil {
		return fmt.Errorf("load server environment: %w", err)
	}

	if err := validation.Validate.Struct(cfg); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	return nil
})
