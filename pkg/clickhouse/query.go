package clickhouse

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func insertQueryWithSettings(insertQuery string, settings map[string]any) (string, error) {
	if len(settings) == 0 {
		return insertQuery, nil
	}

	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value, err := formatInsertSettingValue(settings[key])
		if err != nil {
			return "", fmt.Errorf("setting %q: %w", key, err)
		}

		parts = append(parts, fmt.Sprintf("%s = %s", key, value))
	}

	settingsClause := " SETTINGS " + strings.Join(parts, ", ")

	upper := strings.ToUpper(insertQuery)
	valuesIdx := strings.LastIndex(upper, " VALUES")

	if valuesIdx >= 0 {
		return insertQuery[:valuesIdx] + settingsClause + insertQuery[valuesIdx:], nil
	}

	return insertQuery + settingsClause, nil
}

func formatInsertSettingValue(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return quoteCHString(v), nil
	case bool:
		if v {
			return "1", nil
		}

		return "0", nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("unsupported value type %T", value)
	}
}

func quoteCHString(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}
