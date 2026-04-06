package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fluent/fluent-bit-go/output"
)

// helper to build map[interface{}]interface{} with string keys
func imap(pairs ...interface{}) map[interface{}]interface{} {
	m := make(map[interface{}]interface{})
	for i := 0; i < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return m
}

func TestFormatTimestamp(t *testing.T) {
	t.Run("FLBTime", func(t *testing.T) {
		ts := output.FLBTime{Time: time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)}
		s, ok := formatTimestamp(ts)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if s != "2026-04-06T12:00:00Z" {
			t.Errorf("got %s", s)
		}
	})

	t.Run("uint64", func(t *testing.T) {
		s, ok := formatTimestamp(uint64(1712404800))
		if !ok {
			t.Fatal("expected ok=true")
		}
		if s == "" {
			t.Error("expected non-empty string")
		}
	})

	t.Run("time.Time", func(t *testing.T) {
		s, ok := formatTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		if !ok {
			t.Fatal("expected ok=true")
		}
		if s != "2026-01-01T00:00:00Z" {
			t.Errorf("got %s", s)
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		_, ok := formatTimestamp("not a timestamp")
		if ok {
			t.Error("expected ok=false for string")
		}
	})
}

func TestConvertRecord(t *testing.T) {
	tests := []struct {
		name     string
		input    map[interface{}]interface{}
		expected map[string]interface{}
	}{
		{
			name:  "string keys and byte values",
			input: imap("key1", []byte("value1"), "key2", []byte("value2")),
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:  "mixed types",
			input: imap("str", []byte("hello"), "num", int64(42), "float", float64(3.14), "bool", true, "nilval", nil),
			expected: map[string]interface{}{
				"str":    "hello",
				"num":    int64(42),
				"float":  float64(3.14),
				"bool":   true,
				"nilval": nil,
			},
		},
		{
			name:  "nested map",
			input: imap("outer", imap("inner", []byte("deep"))),
			expected: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": "deep",
				},
			},
		},
		{
			name: "array values",
			input: imap("list", []interface{}{
				[]byte("a"),
				int64(1),
				imap("nested", []byte("value")),
			}),
			expected: map[string]interface{}{
				"list": []interface{}{
					"a",
					int64(1),
					map[string]interface{}{
						"nested": "value",
					},
				},
			},
		},
		{
			name:     "empty map",
			input:    imap(),
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertRecord(tt.input)
			resultJSON, _ := json.Marshal(result)
			expectedJSON, _ := json.Marshal(tt.expected)
			if string(resultJSON) != string(expectedJSON) {
				t.Errorf("got %s, want %s", resultJSON, expectedJSON)
			}
		})
	}
}

func TestRecordToJSON(t *testing.T) {
	cfg := &FlushConfig{
		AddTag:  true,
		TimeKey: "_time",
	}

	t.Run("with FLBTime timestamp", func(t *testing.T) {
		ts := output.FLBTime{Time: time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)}
		record := imap("message", []byte("hello world"))

		jsonBytes, err := recordToJSON(ts, record, "test.tag", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var m map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &m); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		if m["message"] != "hello world" {
			t.Errorf("message = %v, want 'hello world'", m["message"])
		}
		if m["_tag"] != "test.tag" {
			t.Errorf("_tag = %v, want 'test.tag'", m["_tag"])
		}
		if m["_time"] == nil {
			t.Error("_time should be set")
		}
	})

	t.Run("with uint64 timestamp", func(t *testing.T) {
		ts := uint64(1712404800)
		record := imap("level", []byte("info"))

		jsonBytes, err := recordToJSON(ts, record, "app.log", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var m map[string]interface{}
		json.Unmarshal(jsonBytes, &m)

		if m["_time"] == nil {
			t.Error("_time should be set for uint64 timestamp")
		}
		if m["_tag"] != "app.log" {
			t.Errorf("_tag = %v, want 'app.log'", m["_tag"])
		}
	})

	t.Run("no tag", func(t *testing.T) {
		cfgNoTag := &FlushConfig{AddTag: false, TimeKey: "_time"}
		record := imap("msg", []byte("test"))

		jsonBytes, err := recordToJSON(nil, record, "some.tag", cfgNoTag)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var m map[string]interface{}
		json.Unmarshal(jsonBytes, &m)

		if _, exists := m["_tag"]; exists {
			t.Error("_tag should not be set when AddTag is false")
		}
	})

	t.Run("no time key", func(t *testing.T) {
		cfgNoTime := &FlushConfig{AddTag: true, TimeKey: ""}
		ts := output.FLBTime{Time: time.Now()}
		record := imap("msg", []byte("test"))

		jsonBytes, err := recordToJSON(ts, record, "tag", cfgNoTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var m map[string]interface{}
		json.Unmarshal(jsonBytes, &m)

		if _, exists := m["_time"]; exists {
			t.Error("_time should not be set when TimeKey is empty")
		}
	})

	t.Run("existing timestamp not overwritten", func(t *testing.T) {
		ts := output.FLBTime{Time: time.Now()}
		record := imap("_time", []byte("original-time"))

		jsonBytes, err := recordToJSON(ts, record, "", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var m map[string]interface{}
		json.Unmarshal(jsonBytes, &m)

		if m["_time"] != "original-time" {
			t.Errorf("_time = %v, want 'original-time' (should not overwrite)", m["_time"])
		}
	})

	t.Run("existing tag not overwritten", func(t *testing.T) {
		record := imap("_tag", []byte("original-tag"))

		jsonBytes, err := recordToJSON(nil, record, "new.tag", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var m map[string]interface{}
		json.Unmarshal(jsonBytes, &m)

		if m["_tag"] != "original-tag" {
			t.Errorf("_tag = %v, want 'original-tag' (should not overwrite)", m["_tag"])
		}
	})
}

func TestEnsureURLScheme(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "https://example.com"},
		{"http://example.com", "http://example.com"},
		{"example.com", "https://example.com"},
		{"abc123.zerobus.us-west-2.cloud.databricks.com", "https://abc123.zerobus.us-west-2.cloud.databricks.com"},
		{"https://abc123.zerobus.us-west-2.cloud.databricks.com", "https://abc123.zerobus.us-west-2.cloud.databricks.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ensureURLScheme(tt.input)
			if result != tt.expected {
				t.Errorf("ensureURLScheme(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
