package repokeeper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/skaphos/repokeeper/internal/cliio"
	"github.com/spf13/cobra"
)

type outputKind string

const (
	outputKindTable         outputKind = "table"
	outputKindWide          outputKind = "wide"
	outputKindJSON          outputKind = "json"
	outputKindCustomColumns outputKind = "custom-columns"
)

type outputMode struct {
	kind outputKind
	expr string
}

type customColumnSpec struct {
	header string
	expr   string
}

func parseOutputMode(format string) (outputMode, error) {
	trimmed := strings.TrimSpace(format)
	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasPrefix(lower, "custom-columns="):
		expr := strings.TrimSpace(trimmed[len("custom-columns="):])
		if expr == "" {
			return outputMode{}, fmt.Errorf("custom-columns output requires column definitions")
		}
		return outputMode{kind: outputKindCustomColumns, expr: expr}, nil
	case lower == string(outputKindTable), lower == "":
		return outputMode{kind: outputKindTable}, nil
	case lower == string(outputKindWide):
		return outputMode{kind: outputKindWide}, nil
	case lower == string(outputKindJSON):
		return outputMode{kind: outputKindJSON}, nil
	default:
		return outputMode{}, fmt.Errorf("unsupported format %q", format)
	}
}

func writeCustomColumnsOutput(cmd *cobra.Command, output any, spec string, noHeaders bool) error {
	value, err := marshalToGeneric(output)
	if err != nil {
		return err
	}
	columns, err := parseCustomColumnsSpec(spec)
	if err != nil {
		return err
	}
	rows := rowsForCustomColumns(value)
	headers := make([]string, 0, len(columns))
	for _, col := range columns {
		headers = append(headers, col.header)
	}

	renderRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		values := make([]string, 0, len(columns))
		for _, col := range columns {
			cell, err := resolveCustomColumnValue(row, col.expr)
			if err != nil {
				return err
			}
			values = append(values, cell)
		}
		renderRows = append(renderRows, values)
	}
	return cliio.WriteTable(cmd.OutOrStdout(), false, noHeaders, headers, renderRows)
}

func parseCustomColumnsSpec(raw string) ([]customColumnSpec, error) {
	parts := strings.Split(raw, ",")
	columns := make([]customColumnSpec, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		pieces := strings.SplitN(trimmed, ":", 2)
		if len(pieces) != 2 {
			return nil, fmt.Errorf("invalid custom-columns segment %q (expected NAME:JSONPATH)", trimmed)
		}
		header := strings.TrimSpace(pieces[0])
		expr := strings.TrimSpace(pieces[1])
		if header == "" || expr == "" {
			return nil, fmt.Errorf("invalid custom-columns segment %q (expected NAME:JSONPATH)", trimmed)
		}
		if !strings.HasPrefix(expr, ".") {
			expr = "." + expr
		}
		columns = append(columns, customColumnSpec{header: header, expr: expr})
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("custom-columns output requires at least one NAME:JSONPATH pair")
	}
	return columns, nil
}

func rowsForCustomColumns(output any) []any {
	switch typed := output.(type) {
	case []any:
		return typed
	case map[string]any:
		for _, key := range []string{"repos", "results", "items"} {
			if values, ok := typed[key].([]any); ok {
				return values
			}
		}
		return []any{typed}
	default:
		return []any{typed}
	}
}

func marshalToGeneric(input any) (any, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	var output any
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, err
	}
	return output, nil
}

func resolveCustomColumnValue(row any, expr string) (string, error) {
	path := strings.TrimSpace(expr)
	if path == "" || path == "." {
		return "", nil
	}
	path = strings.TrimPrefix(path, ".")
	parts := strings.Split(path, ".")

	current := row
	for _, part := range parts {
		key := strings.TrimSpace(part)
		if key == "" {
			continue
		}
		obj, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("custom-columns path %q is not a map at %q", expr, key)
		}
		next, exists := obj[key]
		if !exists {
			return "", nil
		}
		current = next
	}
	switch value := current.(type) {
	case nil:
		return "", nil
	case string:
		return value, nil
	case bool:
		if value {
			return "true", nil
		}
		return "false", nil
	default:
		return fmt.Sprint(value), nil
	}
}
