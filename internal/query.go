// Copyright 2025 Laurin Heilmeyer. All rights reserved.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package internal

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
)

func NewSelectFunction(conn *pgxpool.Pool) rego.Builtin2 {
	return func(bctx rego.BuiltinContext, queryTerm *ast.Term, argsTerm *ast.Term) (*ast.Term, error) {
		var query string
		if err := ast.As(queryTerm.Value, &query); err != nil {
			return nil, fmt.Errorf("postgres.select: invalid query: %w", err)
		}

		var argsArray []interface{}
		if arr, ok := argsTerm.Value.(*ast.Array); ok {
			for i := 0; i < arr.Len(); i++ {
				var val interface{}
				if err := ast.As(arr.Elem(i).Value, &val); err != nil {
					return nil, fmt.Errorf("postgres.select: invalid argument at position %d: %w", i, err)
				}
				argsArray = append(argsArray, val)
			}
		} else {
			return nil, fmt.Errorf("postgres.select: second argument must be an array")
		}

		rows, err := conn.Query(context.Background(), query, argsArray...)
		if err != nil {
			return nil, fmt.Errorf("postgres.select: query failed: %w", err)
		}
		defer rows.Close()

		var result []map[string]interface{}
		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				return nil, fmt.Errorf("postgres.select: failed to read row: %w", err)
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
			return nil, fmt.Errorf("postgres.select: failed to convert result: %w", err)
		}

		return ast.NewTerm(resultValue), nil
	}
}
