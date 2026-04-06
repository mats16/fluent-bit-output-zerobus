package main

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
)

// PluginConfig holds configuration for the ZeroBus output plugin.
type PluginConfig struct {
	// Required
	ZerobusEndpoint string // gRPC endpoint URL (e.g., "https://<workspace-id>.zerobus.<region>.cloud.databricks.com")
	WorkspaceURL    string // Databricks workspace URL (e.g., "https://<instance>.cloud.databricks.com")
	TableName       string // Fully qualified table name (catalog.schema.table)
	ClientID        string // OAuth2 client ID
	ClientSecret    string // OAuth2 client secret

	// Optional
	AddTag  bool     // Add Fluent Bit tag as _tag field (default: true)
	TimeKey string   // Timestamp field name, empty to disable (default: "_time")
	LogKeys   []string // If non-empty, only these keys are included in the output record
	RawLogKey string   // If non-empty, store the raw record as a JSON string in this field
}

// ensureURLScheme prepends "https://" if the value has no scheme.
func ensureURLScheme(v string) string {
	if strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "http://") {
		return v
	}
	return "https://" + v
}

func parseConfig(plugin unsafe.Pointer) (*PluginConfig, error) {
	cfg := &PluginConfig{
		AddTag:  true,
		TimeKey: "_time",
	}

	cfg.ZerobusEndpoint = output.FLBPluginConfigKey(plugin, "zerobus_endpoint")
	if cfg.ZerobusEndpoint == "" {
		return nil, fmt.Errorf("zerobus_endpoint is required")
	}
	cfg.ZerobusEndpoint = ensureURLScheme(cfg.ZerobusEndpoint)

	cfg.WorkspaceURL = output.FLBPluginConfigKey(plugin, "workspace_url")
	if cfg.WorkspaceURL == "" {
		return nil, fmt.Errorf("workspace_url is required")
	}
	cfg.WorkspaceURL = ensureURLScheme(cfg.WorkspaceURL)

	cfg.TableName = output.FLBPluginConfigKey(plugin, "table_name")
	if cfg.TableName == "" {
		return nil, fmt.Errorf("table_name is required")
	}

	cfg.ClientID = output.FLBPluginConfigKey(plugin, "client_id")
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("client_id is required")
	}

	cfg.ClientSecret = output.FLBPluginConfigKey(plugin, "client_secret")
	if cfg.ClientSecret == "" {
		return nil, fmt.Errorf("client_secret is required")
	}

	if v := output.FLBPluginConfigKey(plugin, "add_tag"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid add_tag: %w", err)
		}
		cfg.AddTag = b
	}

	if v := output.FLBPluginConfigKey(plugin, "time_key"); v != "" {
		cfg.TimeKey = v
	}

	cfg.RawLogKey = output.FLBPluginConfigKey(plugin, "raw_log_key")

	if v := output.FLBPluginConfigKey(plugin, "log_key"); v != "" {
		for _, k := range strings.Split(v, ",") {
			if trimmed := strings.TrimSpace(k); trimmed != "" {
				cfg.LogKeys = append(cfg.LogKeys, trimmed)
			}
		}
	}

	return cfg, nil
}
