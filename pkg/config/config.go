// Copyright 2025 Laurin Heilmeyer. All rights reserved.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package config

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/open-policy-agent/opa/v1/plugins"
	"github.com/open-policy-agent/opa/v1/util"
)

type Config struct {
	ConnectionString string `json:"connection_string,omitempty"`

	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Database string `json:"database,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	SSLMode  string `json:"ssl_mode,omitempty"`

	ConnectTimeoutSeconds int               `json:"connect_timeout_seconds,omitempty"`
	ApplicationName       string            `json:"application_name,omitempty"`
	SearchPath            string            `json:"search_path,omitempty"`
	Options               map[string]string `json:"options,omitempty"`
}

func New(m *plugins.Manager, bs []byte) (*Config, error) {
	cfg := Config{
		Host:     defaultHost,
		Port:     defaultPort,
		Database: defaultDatabase,
	}

	if err := util.Unmarshal(bs, &cfg); err != nil {
		return nil, err
	}

	if cfg.ConnectionString == "" {
		connectionString, err := buildConnectionString(cfg)
		if err != nil {
			return nil, err
		}
		cfg.ConnectionString = connectionString
	}

	return &cfg, nil
}

func buildConnectionString(cfg Config) (string, error) {
	connStr := fmt.Sprintf("postgres://%s:%d/%s", cfg.Host, cfg.Port, cfg.Database)

	params := url.Values{}

	if cfg.User != "" {
		params.Add("user", cfg.User)
	}
	if cfg.Password != "" {
		params.Add("password", cfg.Password)
	}

	if cfg.SSLMode != "" {
		params.Add("sslmode", cfg.SSLMode)
	}

	if cfg.ConnectTimeoutSeconds > 0 {
		params.Add("connect_timeout_seconds", strconv.Itoa(cfg.ConnectTimeoutSeconds))
	}

	if cfg.ApplicationName != "" {
		params.Add("application_name", cfg.ApplicationName)
	}

	if cfg.SearchPath != "" {
		params.Add("search_path", cfg.SearchPath)
	}

	for k, v := range cfg.Options {
		params.Add(k, v)
	}

	if len(params) > 0 {
		connStr = connStr + "?" + params.Encode()
	}

	return connStr, nil
}
