// Copyright 2025 Laurin Heilmeyer. All rights reserved.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"fmt"

	"github.com/heilmela/opa-postgres-plugin/internal"
	cfg "github.com/heilmela/opa-postgres-plugin/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-policy-agent/opa/v1/plugins"
)

var PgxPoolConnect = pgxpool.New
var PluginName = cfg.PluginName

type PostgresPlugin struct {
	manager *plugins.Manager
	config  cfg.Config
	pool    *pgxpool.Pool
}

func (p *PostgresPlugin) Start(ctx context.Context) error {
	logger := p.manager.Logger()

	configLogFields := map[string]interface{}{
		"connection_string": p.config.ConnectionString,
		"connection_params": p.config.ConnectionParams,
	}
	logger.WithFields(configLogFields).Debug("postgres plugin configuration")

	logger.Debug("attempting to connect to database...")

	if p.config.ConnectionString == "" {
		connectionString, err := cfg.BuildConnectionString(p.config.ConnectionParams)
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

	internal.UpdateDatabaseConnection(p.pool)
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
	return
}
