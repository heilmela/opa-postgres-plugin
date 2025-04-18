// Copyright 2025 Laurin Heilmeyer. All rights reserved.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"os"

	cfg "github.com/heilmela/opa-postgres-plugin/pkg/config"
	"github.com/heilmela/opa-postgres-plugin/pkg/plugin"
	"github.com/open-policy-agent/opa/cmd"
	"github.com/open-policy-agent/opa/v1/runtime"
)

func main() {
	runtime.RegisterPlugin(cfg.PluginName, plugin.Factory{})

	if err := cmd.RootCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
