package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluent/fluent-bit-go/output"
)

// formatTimestamp converts a Fluent Bit timestamp to an RFC3339Nano string.
// Returns the formatted string and true if the timestamp was recognized, or ("", false) otherwise.
func formatTimestamp(ts interface{}) (string, bool) {
	switch t := ts.(type) {
	case output.FLBTime:
		return t.Time.UTC().Format(time.RFC3339Nano), true
	case uint64:
		return time.Unix(int64(t), 0).UTC().Format(time.RFC3339Nano), true
	case time.Time:
		return t.UTC().Format(time.RFC3339Nano), true
	default:
		return "", false
	}
}

// convertValue recursively converts Fluent Bit record values to JSON-serializable types.
func convertValue(v interface{}) interface{} {
	switch val := v.(type) {
	case []byte:
		return string(val)
	case map[interface{}]interface{}:
		return convertRecord(val)
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = convertValue(item)
		}
		return result
	case output.FLBTime, time.Time:
		if s, ok := formatTimestamp(val); ok {
			return s
		}
		return val
	default:
		return val
	}
}

// convertRecord transforms a Fluent Bit record (map[interface{}]interface{} with []byte keys)
// into a JSON-serializable map[string]interface{}.
func convertRecord(record map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(record))
	for k, v := range record {
		var key string
		switch kt := k.(type) {
		case []byte:
			key = string(kt)
		case string:
			key = kt
		default:
			key = fmt.Sprintf("%v", kt)
		}
		result[key] = convertValue(v)
	}
	return result
}

// recordToJSON converts a Fluent Bit record to a JSON byte slice for ZeroBus ingestion.
// It injects timestamp and tag fields based on the plugin configuration.
func recordToJSON(ts interface{}, record map[interface{}]interface{}, tag string, cfg *FlushConfig) ([]byte, error) {
	m := convertRecord(record)

	var rawJSON string
	if cfg.RawLogKey != "" {
		raw, err := json.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal raw record: %w", err)
		}
		rawJSON = string(raw)
	}

	if len(cfg.LogKeys) > 0 {
		filtered := make(map[string]interface{}, len(cfg.LogKeys))
		for _, k := range cfg.LogKeys {
			if v, exists := m[k]; exists {
				filtered[k] = v
			}
		}
		m = filtered
	}

	if cfg.RawLogKey != "" {
		m[cfg.RawLogKey] = rawJSON
	}

	if cfg.TimeKey != "" {
		if _, exists := m[cfg.TimeKey]; !exists {
			if s, ok := formatTimestamp(ts); ok {
				m[cfg.TimeKey] = s
			}
		}
	}

	if cfg.AddTag && tag != "" {
		if _, exists := m["_tag"]; !exists {
			m["_tag"] = tag
		}
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record to JSON: %w", err)
	}
	return data, nil
}
