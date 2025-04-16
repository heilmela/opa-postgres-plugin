// Copyright 2025 Laurin Heilmeyer. All rights reserved.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"sync"

	"github.com/heilmela/opa-postgres-plugin/internal"
	cfg "github.com/heilmela/opa-postgres-plugin/pkg/config"
	"github.com/jackc/pgx/v5"
	"github.com/open-policy-agent/opa/v1/plugins"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/types"
)

var PgxConnect = pgx.Connect

type PostgresPlugin struct {
	manager    *plugins.Manager
	mtx        sync.Mutex
	config     cfg.Config
	connection *pgx.Conn
}

func (p *PostgresPlugin) Start(ctx context.Context) error {

	conn, err := PgxConnect(context.Background(), p.config.ConnectionString)
	if err != nil {
		p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{State: plugins.StateNotReady})
		return err
	}
	p.connection = conn

	selectFunction := internal.NewSelectFunction(p.connection)
	rego.RegisterBuiltin2(
		&rego.Function{
			Name:             "postgres.select",
			Decl:             types.NewFunction(types.Args(types.S, types.NewArray([]types.Type{}, types.A)), types.A),
			Memoize:          true,
			Nondeterministic: true,
		},
		selectFunction,
	)

	p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{State: plugins.StateOK})
	return nil
}

func (p *PostgresPlugin) Stop(ctx context.Context) {
	if p.connection != nil {
		p.connection.Close(context.Background())
	}
	p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{State: plugins.StateNotReady})
}

func (p *PostgresPlugin) Reconfigure(ctx context.Context, config interface{}) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.config = config.(cfg.Config)

	if p.connection != nil {
		p.connection.Close(context.Background())
	}

	conn, err := PgxConnect(context.Background(), p.config.ConnectionString)
	if err != nil {
		p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{State: plugins.StateNotReady})
		return
	}
	p.connection = conn

	p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{State: plugins.StateOK})
}
