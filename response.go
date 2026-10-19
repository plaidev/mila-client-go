package mila

import "fmt"

// ConvertRawCellToObj は STRUCT/ARRAY の値をスキーマに沿って再帰的に変換する。
func ConvertRawCellToObj(raw any, schema ColumnSpec) (any, error) {
	if raw == nil {
		return nil, nil
	}
	switch schema.Type {
	case ColumnTypeStruct:
		row, ok := raw.([]any)
		if !ok {
			return nil, fmt.Errorf("mila: STRUCT column %q requires an array, got %T", schema.Name, raw)
		}
		object, err := ConvertRawRowToMap(row, schema.Fields)
		if err != nil {
			return nil, err
		}
		// TypeScript 版と同様、STRUCT 内の null フィールドは省略する。
		for key, value := range object {
			if value == nil {
				delete(object, key)
			}
		}
		return object, nil
	case ColumnTypeArray:
		values, ok := raw.([]any)
		if !ok {
			return nil, fmt.Errorf("mila: ARRAY column %q requires an array, got %T", schema.Name, raw)
		}
		if len(schema.Fields) != 1 {
			return nil, fmt.Errorf("mila: ARRAY column %q requires exactly one element schema", schema.Name)
		}
		result := make([]any, len(values))
		for index, value := range values {
			converted, err := ConvertRawCellToObj(value, schema.Fields[0])
			if err != nil {
				return nil, fmt.Errorf("mila: array element %d: %w", index, err)
			}
			result[index] = converted
		}
		return result, nil
	default:
		return raw, nil
	}
}

// ConvertRawRowToMap はトップレベルの null を保持し、カラム名をキーに変換する。
func ConvertRawRowToMap(raw []any, schema []ColumnSpec) (map[string]any, error) {
	if len(raw) != len(schema) {
		return nil, fmt.Errorf("mila: row has %d cells, schema has %d columns", len(raw), len(schema))
	}
	result := make(map[string]any, len(schema))
	for index, column := range schema {
		value, err := ConvertRawCellToObj(raw[index], column)
		if err != nil {
			return nil, fmt.Errorf("mila: column %q: %w", column.Name, err)
		}
		if column.Name != "" {
			result[column.Name] = value
		}
	}
	return result, nil
}

func ConvertRunSQLRows(raw [][]any, schema []ColumnSpec) ([]map[string]any, error) {
	result := make([]map[string]any, len(raw))
	for index, row := range raw {
		converted, err := ConvertRawRowToMap(row, schema)
		if err != nil {
			return nil, fmt.Errorf("mila: row %d: %w", index, err)
		}
		result[index] = converted
	}
	return result, nil
}
