# fluent-bit-output-zerobus

A [Fluent Bit](https://fluentbit.io/) output plugin that streams log records into Databricks Delta tables via [ZeroBus Ingest](https://docs.databricks.com/aws/en/ingestion/zerobus-overview) gRPC.

## Prerequisites

- Go 1.21+
- CGO enabled (required for both Fluent Bit Go plugins and ZeroBus SDK)
- A Databricks workspace with ZeroBus enabled
- A service principal with `MODIFY` and `SELECT` permissions on the target table

## Build

```bash
make build
```

This produces `out_zerobus.so` which can be loaded by Fluent Bit.

## Configuration

| Key | Required | Default | Description |
|-----|----------|---------|-------------|
| `zerobus_endpoint` | Yes | - | ZeroBus gRPC endpoint URL (e.g., `https://<workspace-id>.zerobus.<region>.cloud.databricks.com`) |
| `workspace_url` | Yes | - | Databricks workspace URL (e.g., `https://<instance>.cloud.databricks.com`) |
| `table_name` | Yes | - | Fully qualified Unity Catalog table name (`catalog.schema.table`) |
| `client_id` | Yes | - | Service principal Application ID |
| `client_secret` | Yes | - | Service principal secret |
| `add_tag` | No | `true` | Add Fluent Bit tag as `_tag` field |
| `time_key` | No | `_time` | Timestamp field name (empty string to disable) |

> **Note:** If `zerobus_endpoint` or `workspace_url` is provided without an `https://` prefix, the plugin will automatically prepend it.

### Example

```ini
[OUTPUT]
    Name              zerobus
    Match             *
    zerobus_endpoint  https://${ZEROBUS_SHARD_ID}.zerobus.${AWS_REGION}.cloud.databricks.com
    workspace_url     https://${DATABRICKS_HOST}
    table_name        catalog.schema.logs
    client_id         ${DATABRICKS_CLIENT_ID}
    client_secret     ${DATABRICKS_CLIENT_SECRET}
```

## Docker

```bash
make docker

docker run -e ZEROBUS_SHARD_ID=... \
           -e AWS_REGION=us-west-2 \
           -e DATABRICKS_HOST=... \
           -e ZEROBUS_TABLE_NAME=catalog.schema.logs \
           -e DATABRICKS_CLIENT_ID=... \
           -e DATABRICKS_CLIENT_SECRET=... \
           fluent-bit-zerobus
```

## How It Works

1. Fluent Bit collects log records from configured inputs
2. This plugin converts each record to JSON
3. All records in a Fluent Bit chunk are sent to ZeroBus as a single atomic batch via the official Go SDK (gRPC)
4. ZeroBus streams the data directly into the specified Delta table
5. The plugin waits for server acknowledgment before confirming to Fluent Bit

## License

Apache License 2.0
