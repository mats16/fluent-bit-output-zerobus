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
    zerobus_endpoint  ${ZEROBUS_ENDPOINT}
    table_name        ${ZEROBUS_TABLE_NAME}
    workspace_url     ${DATABRICKS_HOST}
    client_id         ${DATABRICKS_CLIENT_ID}
    client_secret     ${DATABRICKS_CLIENT_SECRET}
```

## Docker

```bash
make docker
```

### Quick Start

1. Create the destination Delta table.

```sql
CREATE TABLE IF NOT EXISTS main.default.zerobus_fluent_bit (
  message STRING,
  level STRING,
  _time TIMESTAMP,
  _tag STRING
);
```

> **Note:** The schema above matches the built-in dummy input. Adjust columns to match your actual input data.

2. Run the Docker image. By default, the dummy input sends a test record every 5 seconds.

```bash
docker run --rm -it \
  -e ZEROBUS_ENDPOINT=<workspace-id>.zerobus.<region>.cloud.databricks.com \
  -e ZEROBUS_TABLE_NAME=main.default.zerobus_fluent_bit \
  -e DATABRICKS_HOST=https://<instance>.cloud.databricks.com \
  -e DATABRICKS_CLIENT_ID=<service-principal-client-id> \
  -e DATABRICKS_CLIENT_SECRET=<service-principal-secret> \
  -it fluent-bit-zerobus
```

3. Verify that records are being written to the table.

```sql
SELECT * FROM main.default.zerobus_fluent_bit ORDER BY _time DESC LIMIT 10;
```

## How It Works

1. Fluent Bit collects log records from configured inputs
2. This plugin converts each record to JSON
3. All records in a Fluent Bit chunk are sent to ZeroBus as a single atomic batch via the official Go SDK (gRPC)
4. ZeroBus streams the data directly into the specified Delta table
5. The plugin waits for server acknowledgment before confirming to Fluent Bit

## License

Apache License 2.0
