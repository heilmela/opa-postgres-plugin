// Copyright 2025 Laurin Heilmeyer. All rights reserved.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/heilmela/opa-postgres-plugin/internal"
	cfg "github.com/heilmela/opa-postgres-plugin/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-policy-agent/opa/v1/plugins"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/types"
)

var PgxPoolConnect = pgxpool.New

// redundant but nicer to consume for ppl importing the plugin
var PluginName = cfg.PluginName

type PostgresPlugin struct {
	manager *plugins.Manager
	mtx     sync.Mutex
	config  cfg.Config
	pool    *pgxpool.Pool
}

func (p *PostgresPlugin) Start(ctx context.Context) error {
	logger := p.manager.Logger()
	logger.WithFields(map[string]interface{}{
		"host":                  p.config.Host,
		"port":                  p.config.Port,
		"password":              p.config.Password,
		"database":              p.config.Database,
		"user":                  p.config.User,
		"ssl_mode":              p.config.SSLMode,
		"connect_timeout":       p.config.ConnectTimeoutSeconds,
		"application_name":      p.config.ApplicationName,
		"search_path":           p.config.SearchPath,
		"has_connection_string": p.config.ConnectionString != "",
		"has_custom_options":    len(p.config.Options) > 0,
	}).Debug("postgres plugin configuration")

	logger.Debug("attempting to connect to PostgreSQL database...")

	if p.config.ConnectionString == "" {
		connectionString, err := cfg.BuildConnectionString(p.config)
		if err != nil {
			p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{
				State:   plugins.StateNotReady,
				Message: err.Error(),
			})
			logger.WithFields(map[string]interface{}{
				"err": err,
			}).Error("failed to build connection string from config parameters")
			return err
		}
		p.config.ConnectionString = connectionString
	}

	pool, err := PgxPoolConnect(ctx, p.config.ConnectionString)
	if err != nil {
		statusErr := fmt.Sprintf("unable to create connection pool: %v", err)
		p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{
			State:   plugins.StateNotReady,
			Message: statusErr,
		})
		logger.WithFields(map[string]interface{}{
			"err": err,
		}).Error("failed to create postgres connection pool")
		return err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		statusErr := fmt.Sprintf("connection pool created but ping failed: %v", err)
		p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{
			State:   plugins.StateNotReady,
			Message: statusErr,
		})
		logger.WithFields(map[string]interface{}{
			"err": err,
		}).Error("postgres connection pool ping failed")
		return err
	}

	p.pool = pool
	logger.Info("successfully connected to postgres")

	selectFunction := internal.NewSelectFunction(p.pool)
	rego.RegisterBuiltin2(
		&rego.Function{
			Name:             "postgres.select",
			Decl:             types.NewFunction(types.Args(types.S, types.NewArray([]types.Type{}, types.A)), types.A),
			Memoize:          true,
			Nondeterministic: true,
		},
		selectFunction,
	)
	logger.Info("registered postgres.select builtin")

	p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{State: plugins.StateOK})
	return nil
}

func (p *PostgresPlugin) Stop(ctx context.Context) {
	logger := p.manager.Logger()

	if p.pool != nil {
		logger.Info("closing postgres connection pool")
		p.pool.Close()
	}

	p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{
		State: plugins.StateNotReady,
	})
	logger.Info("postgres plugin stopped")
}

func (p *PostgresPlugin) Reconfigure(ctx context.Context, config interface{}) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	logger := p.manager.Logger()
	logger.Info("reconfiguring postgres plugin")

	p.config = config.(cfg.Config)

	if p.pool != nil {
		logger.Info("closing existing postgres connection pool")
		p.pool.Close()
	}

	if p.config.ConnectionString == "" {
		connectionString, err := cfg.BuildConnectionString(p.config)
		if err != nil {
			statusErr := fmt.Sprintf("failed to build connection string: %v", err)
			p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{
				State:   plugins.StateNotReady,
				Message: statusErr,
			})
			logger.WithFields(map[string]interface{}{
				"err": err,
			}).Error("failed to build connection string during reconfiguration")
			return
		}
		p.config.ConnectionString = connectionString
	}

	pool, err := PgxPoolConnect(ctx, p.config.ConnectionString)
	if err != nil {
		statusErr := fmt.Sprintf("unable to create connection pool: %v", err)
		p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{
			State:   plugins.StateNotReady,
			Message: statusErr,
		})
		logger.WithFields(map[string]interface{}{
			"err": err,
		}).Error("failed to create PostgreSQL connection pool during reconfiguration")
		return
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		statusErr := fmt.Sprintf("connection pool created but ping failed: %v", err)
		p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{
			State:   plugins.StateNotReady,
			Message: statusErr,
		})
		logger.WithFields(map[string]interface{}{
			"err": err,
		}).Error("postgres connection pool ping failed during reconfiguration")
		return
	}

	p.pool = pool
	logger.Info("successfully reconfigured postgres connection")

	p.manager.UpdatePluginStatus(cfg.PluginName, &plugins.Status{State: plugins.StateOK})
}
