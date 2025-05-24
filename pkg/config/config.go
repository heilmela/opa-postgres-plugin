// Copyright 2025 Laurin Heilmeyer. All rights reserved.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/open-policy-agent/opa/v1/plugins"
)

type Config struct {
	ConnectionString string            `json:"connection_string,omitempty"`
	ConnectionParams map[string]string `json:"-"`
}

func New(m *plugins.Manager, bs []byte) (*Config, error) {
	cfg := Config{
		ConnectionParams: make(map[string]string),
	}

	var rawConfig map[string]interface{}
	if err := json.Unmarshal(bs, &rawConfig); err != nil {
		return nil, fmt.Errorf("error unmarshalling configuration: %w", err)
	}

	for key, value := range rawConfig {
		if key == "connection_string" {
			if strVal, ok := value.(string); ok {
				cfg.ConnectionString = strVal
			} else if value != nil {
				return nil, fmt.Errorf("configuration key 'connection_string' must be a string, got %T", value)
			}
		} else if key == "connection_params" {
			if paramsMap, ok := value.(map[string]interface{}); ok {
				for paramName, paramValue := range paramsMap {
					var strVal string
					switch v := paramValue.(type) {
					case string:
						strVal = v
					case float64:
						if v == float64(int64(v)) {
							strVal = fmt.Sprintf("%d", int64(v))
						} else {
							strVal = fmt.Sprintf("%g", v)
						}
					case bool:
						strVal = fmt.Sprintf("%t", v)
					case nil:
						continue
					default:
						return nil, fmt.Errorf("unsupported value type for connection_params key '%s': %T. Value must be a string, number, boolean, or null", paramName, paramValue)
					}
					cfg.ConnectionParams[paramName] = strVal
				}
			} else if value != nil {
				return nil, fmt.Errorf("configuration key 'connection_params' must be an object, got %T", value)
			}
		}
	}

	if cfg.ConnectionString == "" {
		if len(cfg.ConnectionParams) > 0 {
			connectionString, err := BuildConnectionString(cfg.ConnectionParams)
			if err != nil {
				return nil, fmt.Errorf("error building connection string from options: %w", err)
			}
			cfg.ConnectionString = connectionString
		} else {

			return nil, fmt.Errorf("no 'connection_string' provided and no parameters found in 'connection_params' to build one")
		}
	}

	return &cfg, nil
}

func BuildConnectionString(options map[string]string) (string, error) {
	if len(options) == 0 {
		return "postgresql:///", nil
	}

	u := &url.URL{
		Scheme: "postgresql",
		Path:   "/",
	}

	params := url.Values{}
	for k, v := range options {
		params.Add(k, v)
	}
	u.RawQuery = params.Encode()

	return u.String(), nil
}
