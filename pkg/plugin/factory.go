// Copyright 2025 Laurin Heilmeyer. All rights reserved.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package plugin

import (
	cfg "github.com/heilmela/opa-postgres-plugin/pkg/config"
	"github.com/open-policy-agent/opa/v1/plugins"
)

type Factory struct{}

func (Factory) New(m *plugins.Manager, config interface{}) plugins.Plugin {
	m.UpdatePluginStatus(cfg.PluginName, &plugins.Status{State: plugins.StateNotReady})
	parsedConfig, ok := config.(*cfg.Config)
	if !ok {
		m.UpdatePluginStatus(cfg.PluginName, &plugins.Status{
			State:   plugins.StateErr,
			Message: "Invalid plugin configuration type",
		})
		return nil
	}

	plugin := &PostgresPlugin{
		manager: m,
		config:  *parsedConfig,
	}

	return plugin
}

func (Factory) Validate(m *plugins.Manager, config []byte) (interface{}, error) {
	parsedConfig, err := cfg.New(m, config)
	return parsedConfig, err
}
