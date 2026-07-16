// Copyright 2024 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package server

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/immanent-tech/go-base/config"
	"github.com/immanent-tech/go-base/validation"
)

const (
	configEnvPrefix         = "WWW"
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
	Port         uint64          `koanf:"port"         validate:"port"`
	Host         string          `koanf:"host"         validate:"hostname|fqdn|ip"`
	CertFile     string          `koanf:"crt"          validate:"omitempty,file"`
	KeyFile      string          `koanf:"key"          validate:"omitempty,file"`
	ReadTimeout  config.Duration `koanf:"readtimeout"  validate:"required"`
	WriteTimeout config.Duration `koanf:"writetimeout" validate:"required"`
	IdleTimeout  config.Duration `koanf:"idletimeout"  validate:"required"`
}

// loadConfigOnce loads the server configuration and ensures this is only done
// one time, no matter how many times it is called.
var loadConfigOnce = sync.OnceValue(func() error {
	// Load server config.
	if err := config.Load(configEnvPrefix, &cfg); err != nil {
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
