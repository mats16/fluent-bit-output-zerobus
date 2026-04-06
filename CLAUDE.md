# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Fluent Bit output plugin (Go, c-shared) that streams log records into Databricks Delta tables via the ZeroBus Go SDK (gRPC). The plugin converts Fluent Bit records to JSON and sends them atomically per flush chunk.

## Build & Test

```bash
make build     # CGO_ENABLED=1 go build -buildmode=c-shared -o out_zerobus.so .
make test      # go test -v -count=1 ./...
make docker    # docker build -t fluent-bit-zerobus .
make clean     # rm out_zerobus.so out_zerobus.h
```

CGO is required — both the Fluent Bit Go plugin (`-buildmode=c-shared`) and the ZeroBus SDK (Rust FFI static link) depend on it. The SDK ships prebuilt static libraries for linux/{amd64,arm64} and darwin/{amd64,arm64}; no Rust toolchain is needed.

## Architecture

All source files are `package main` (required by `-buildmode=c-shared`).

- **main.go** — Four `//export` entry points that Fluent Bit calls: `FLBPluginRegister`, `FLBPluginInit`, `FLBPluginFlushCtx`, `FLBPluginExit`. A `sync.Map` tracks all `ZeroBusClient` instances for global cleanup since `FLBPluginExit` has no per-instance context.
- **config.go** — `parseConfig` reads Fluent Bit config keys via `output.FLBPluginConfigKey`. `PluginConfig` holds connection params (used only at init); `FlushConfig` holds the subset needed per-flush (no secrets retained after init). `ensureURLScheme` normalizes bare hostnames to `https://` URLs as the SDK requires.
- **convert.go** — `convertRecord` recursively transforms `map[interface{}]interface{}` (msgpack-decoded) to `map[string]interface{}` (JSON-serializable). `recordToJSON` adds timestamp/tag fields and marshals to `[]byte`. `formatTimestamp` is the single source of truth for timestamp→string conversion.
- **zerobus.go** — `ZeroBusClient` wraps `ZerobusSdk` + `ZerobusStream`. `Ingest` sends all records via `IngestRecordsOffset` (all-or-nothing) and blocks on `WaitForOffset`. `IsRetryable` uses `errors.As` to check `ZerobusError.Retryable()`.

## Key Design Decisions

- **No sub-batching**: The entire Fluent Bit flush chunk is sent as a single `IngestRecordsOffset` call. This preserves atomicity — partial commits would cause duplicates on retry since Fluent Bit retries whole chunks.
- **Decoder error handling**: `GetRecord` ret == -1 is EOF; ret < -1 (-2 through -5) are malformed records that are skipped and counted. If all records are malformed, returns `FLB_ERROR`. If some succeed, sends them and returns `FLB_OK`.
- **JSON record type**: The SDK is configured with `RecordTypeJson` to avoid per-table protobuf schema generation. Records are sent as JSON strings.

## Dependencies

- `github.com/fluent/fluent-bit-go` — Go bindings for Fluent Bit plugin interface
- `github.com/databricks/zerobus-sdk/go` — ZeroBus Go SDK (CGO wrapper around Rust FFI)
