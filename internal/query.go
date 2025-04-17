package internal

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/topdown/print"
	"github.com/open-policy-agent/opa/v1/types"
)

var (
	dbPool  *pgxpool.Pool
	dbMutex sync.RWMutex
)

func UpdateDatabaseConnection(conn *pgxpool.Pool) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	dbPool = conn
}

func GetDatabaseConnection() *pgxpool.Pool {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return dbPool
}

// Register the builtin function during package initialization
// otherwise policies will fail with undefined function
func init() {
	rego.RegisterBuiltin2(
		&rego.Function{
			Name:             "postgres.query",
			Decl:             types.NewFunction(types.Args(types.S, types.NewArray([]types.Type{}, types.A)), types.A),
			Memoize:          true,
			Nondeterministic: true,
		},
		func(bctx rego.BuiltinContext, queryTerm, argsTerm *ast.Term) (*ast.Term, error) {
			conn := GetDatabaseConnection()
			pctx := print.Context{
				Context:  bctx.Context,
				Location: bctx.Location,
			}

			if conn == nil {
				err := bctx.PrintHook.Print(pctx, "postgres.query: database connection not yet established")
				return nil, err
			}

			var query string
			if err := ast.As(queryTerm.Value, &query); err != nil {
				err = bctx.PrintHook.Print(pctx, fmt.Sprintf("postgres.query: invalid query: %v", err))
				return nil, err
			}

			var argsArray []interface{}
			if arr, ok := argsTerm.Value.(*ast.Array); ok {
				for i := 0; i < arr.Len(); i++ {
					var val interface{}
					if err := ast.As(arr.Elem(i).Value, &val); err != nil {
						err = bctx.PrintHook.Print(pctx, fmt.Sprintf("postgres.query: invalid argument at position %d: %v", i, err))
						return nil, err
					}
					argsArray = append(argsArray, val)
				}
			} else {
				err := bctx.PrintHook.Print(pctx, "postgres.query: second argument must be an array")
				return nil, err
			}

			rows, err := conn.Query(context.Background(), query, argsArray...)
			if err != nil {
				err = bctx.PrintHook.Print(pctx, fmt.Sprintf("postgres.query: query failed: %v", err))
				return nil, err
			}
			defer rows.Close()

			var result []map[string]interface{}
			for rows.Next() {
				values, err := rows.Values()
				if err != nil {
					err = bctx.PrintHook.Print(pctx, fmt.Sprintf("postgres.query: failed to read row: %v", err))
					return nil, err
				}

				fieldDescriptions := rows.FieldDescriptions()
				row := make(map[string]interface{})

				for i, val := range values {
					row[string(fieldDescriptions[i].Name)] = val
				}

				result = append(result, row)
			}

			resultValue, err := ast.InterfaceToValue(result)
			if err != nil {
				err = bctx.PrintHook.Print(pctx, fmt.Sprintf("postgres.query: failed to convert result: %v", err))
				return nil, err
			}

			return ast.NewTerm(resultValue), nil
		},
	)
}
