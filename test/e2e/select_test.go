package e2e_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/runtime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	cfg "github.com/heilmela/opa-postgres-plugin/pkg/config"
	"github.com/heilmela/opa-postgres-plugin/pkg/plugin"
)

func TestPostgresPlugin(t *testing.T) {
	ctx := context.Background()

	pgContainer, pgConnString := startPostgresContainer(t, ctx)
	defer pgContainer.Terminate(ctx)

	conn, err := pgx.Connect(ctx, pgConnString)
	require.NoError(t, err)
	defer conn.Close(ctx)
	seedDatabase(t, ctx, conn)

	runtime.RegisterPlugin(cfg.PluginName, plugin.Factory{})

	params := runtime.NewParams()
	params.ConfigOverrides = []string{
		"plugins." + cfg.PluginName + ".connection_string=" + pgConnString,
	}

	rt, err := runtime.NewRuntime(ctx, params)

	require.NoError(t, err)
	defer rt.Manager.Stop(ctx)

	registeredPlugins := rt.Manager.Plugins()
	t.Logf("Registered plugins: %v", registeredPlugins)

	pluginFound := false
	for _, p := range registeredPlugins {
		if p == cfg.PluginName {
			pluginFound = true
			break
		}
	}
	require.True(t, pluginFound, "Plugin %s should be registered", cfg.PluginName)

	status := rt.Manager.PluginStatus()
	t.Logf("Plugin status: %+v", status)

	if err := rt.Manager.Start(ctx); err != nil {
		t.Fatalf("Failed to start plugin: %v", err)
	}

	policyPath := filepath.Join("..", "testdata", "policies", "authz.rego")
	policyBytes, err := os.ReadFile(policyPath)
	require.NoError(t, err, "Failed to read policy file")
	policyContent := string(policyBytes)

	r := rego.New(
		rego.Query("data.authz.allow"),
		rego.Module("authz.rego", policyContent),
	)

	testCases := []struct {
		name     string
		input    map[string]interface{}
		expected bool
	}{
		{
			name: "User has access to room",
			input: map[string]interface{}{
				"user_id": "user1",
				"room_id": "room1",
			},
			expected: true,
		},
		{
			name: "User does not have access to room",
			input: map[string]interface{}{
				"user_id": "user2",
				"room_id": "room3",
			},
			expected: false,
		},
		{
			name: "Non-existent user",
			input: map[string]interface{}{
				"user_id": "nonexistent",
				"room_id": "room1",
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			preparedQuery, err := r.PrepareForEval(ctx)
			require.NoError(t, err, "Failed to prepare query")

			evalResult, err := preparedQuery.Eval(ctx, rego.EvalInput(tc.input))
			require.NoError(t, err, "Failed to evaluate query")

			allowed := len(evalResult) > 0 && evalResult[0].Expressions[0].Value == true
			assert.Equal(t, tc.expected, allowed, "Unexpected authorization result")
		})
	}
}

func startPostgresContainer(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
	t.Helper()

	pgPort := "5432/tcp"
	dbName := "testdb"
	dbUser := "postgres"
	dbPassword := "postgres"

	req := testcontainers.ContainerRequest{
		Image:        "postgres:14-alpine",
		ExposedPorts: []string{pgPort},
		Env: map[string]string{
			"POSTGRES_DB":       dbName,
			"POSTGRES_USER":     dbUser,
			"POSTGRES_PASSWORD": dbPassword,
		},
		WaitingFor: wait.ForListeningPort(nat.Port(pgPort)).WithStartupTimeout(time.Second * 30),
	}

	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Failed to start postgres container")

	host, err := pgContainer.Host(ctx)
	require.NoError(t, err)

	mappedPort, err := pgContainer.MappedPort(ctx, nat.Port(pgPort))
	require.NoError(t, err)

	pgConnString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		dbUser, dbPassword, host, mappedPort.Port(), dbName)

	time.Sleep(2 * time.Second)

	return pgContainer, pgConnString
}

func seedDatabase(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()

	seedPath := filepath.Join("..", "testdata", "seed", "rooms.sql")
	seedSQL, err := os.ReadFile(seedPath)
	require.NoError(t, err, "Failed to read seed SQL file")

	_, err = conn.Exec(ctx, string(seedSQL))
	require.NoError(t, err, "Failed to execute seed SQL")
}
