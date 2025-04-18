# opa-postgres-plugin

This repository contains a minimal OPA plugin which extends OPA with an interface to pgx for querying PostgreSQL at runtime.

## Overview

The `opa-postgres-plugin` allows you to execute PostgreSQL queries directly from your Rego policies. This enables you to make data-driven policy decisions based on information stored in your PostgreSQL database.

## Features

- Execute SQL queries from within Rego policies
- Pass parameters to queries safely
- Return query results as structured data for use in policy evaluation

## Installation

### As a CLI tool

```bash
go install github.com/heilmela/opa-postgres-plugin@latest
```
### As a Library

```bash
go get github.com/heilmela/opa-postgres-plugin
```

See `main.go` or the [OPA Documentation](https://www.openpolicyagent.org/docs/latest/extensions/#custom-built-in-function-in-go) on how to add plugins to opa.

## Usage

### Configuration

Configure the plugin in your OPA configuration file. You can use either a connection string or individual parameters:

#### Option 1: Using a connection string

```yaml
plugins:
  postgres:
    connection_string: postgres://username:password@localhost:5432/database
```    
### Option 2: Using individual parameters

```yaml
plugins:
  postgres:
    host: localhost
    port: 5432
    database: mydatabase
    user: username
    password: password
    ssl_mode: prefer  # Options: disable, prefer, require, verify-ca, verify-full
```

### Additional configuration options
```yaml
plugins:
  postgres:
    # Basic connection (using either connection_string or individual parameters)

    # Advanced options
    connect_timeout: 10  # seconds
    application_name: my-opa-app
    search_path: public,custom_schema
    options:
      statement_timeout: 5000  # milliseconds
      idle_in_transaction_session_timeout: 10000  # milliseconds

# Default values:
# - host: "localhost"
# - port: 5432
# - database: "postgres"
# - ssl_mode: "prefer"
# - connect_timeout: 10 seconds
# - application_name: "opa-postgres-plugin"
```

### In Rego Policies

Once configured, you can use the `postgres.query` function in your Rego policies:

```rego
package example

import future.keywords.if

default allow := false

# Check if user has required permission
allow if {
  # Get user&#x27;s roles from database
  user_id := input.user.id
  roles := postgres.query(SELECT role FROM user_roles WHERE user_id = $1, [user_id])

  # Check if user has admin role
  some i
  roles[i].role == admin
}
```

## API Reference

### `postgres.query(query, args)`

Executes a SQL query against the configured PostgreSQL database.

**Parameters:**
- `query` (string): SQL query with positional parameters ($1, $2, etc.)
- `args` (array): Array of arguments to pass to the query

**Returns:**
- Array of objects, where each object represents a row with column names as keys

## Building from Source

```bash
# Clone the repository
git clone https://github.com/heilmela/opa-postgres-plugin.git
cd opa-postgres-plugin

# Build the plugin
go build -o opa-postgres ./cmd/main.go
```

## Project Structure

This project is structured to minimize dependencies for users who only want to use parts of the functionality:

- `pkg/`: Core plugin functionality
- `internal/`: Implementation details
- `cmd/`: Command-line tools and container entry points

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
