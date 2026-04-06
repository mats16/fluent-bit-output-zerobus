# fluent-bit-output-zerobus

[Fluent Bit](https://fluentbit.io/) のアウトプットプラグインで、[ZeroBus Ingest](https://docs.databricks.com/aws/en/ingestion/zerobus-overview) gRPC を介してログレコードを Databricks Delta テーブルにストリーミングします。

## 前提条件

- Go 1.21+
- CGO 有効（Fluent Bit Go プラグインと ZeroBus SDK の両方に必要）
- ZeroBus が有効な Databricks ワークスペース
- 対象テーブルに対する `MODIFY` および `SELECT` 権限を持つサービスプリンシパル

## ビルド

```bash
make build
```

`out_zerobus.so` が生成され、Fluent Bit から読み込めます。

## 設定

| キー | 必須 | デフォルト | 説明 |
|------|------|-----------|------|
| `zerobus_endpoint` | はい | - | ZeroBus gRPC エンドポイント URL（例: `https://<workspace-id>.zerobus.<region>.cloud.databricks.com`） |
| `workspace_url` | はい | - | Databricks ワークスペース URL（例: `https://<instance>.cloud.databricks.com`） |
| `table_name` | はい | - | Unity Catalog の完全修飾テーブル名（`catalog.schema.table`） |
| `client_id` | はい | - | サービスプリンシパルのアプリケーション ID |
| `client_secret` | はい | - | サービスプリンシパルのシークレット |
| `add_tag` | いいえ | `true` | Fluent Bit のタグを `_tag` フィールドとして追加 |
| `time_key` | いいえ | `_time` | タイムスタンプのフィールド名（空文字で無効化） |
| `log_key` | いいえ | - | 出力レコードに含めるキーのカンマ区切りリスト。未指定の場合は全フィールドを送信。 |
| `raw_log_key` | いいえ | - | 指定すると、`log_key` フィルタ適用前の元レコード全体を JSON 文字列としてこのフィールドに格納。 |

> **注意:** `zerobus_endpoint` または `workspace_url` に `https://` プレフィックスが付いていない場合、プラグインが自動的に付与します。

### 設定例

```ini
[OUTPUT]
    Name              zerobus
    Match             *
    zerobus_endpoint  ${ZEROBUS_ENDPOINT}
    table_name        ${ZEROBUS_TABLE_NAME}
    workspace_url     ${DATABRICKS_HOST}
    client_id         ${DATABRICKS_CLIENT_ID}
    client_secret     ${DATABRICKS_CLIENT_SECRET}
    log_key           message, level
    raw_log_key       _raw
```

## Docker

```bash
make docker
```

### クイックスタート

1. 送信先の Delta テーブルを作成します。

```sql
CREATE TABLE IF NOT EXISTS main.default.zerobus_fluent_bit (
  message STRING,
  level STRING,
  _raw STRING,
  _time TIMESTAMP,
  _tag STRING
);
```

> **注意:** 上記のスキーマは `log_key` と `raw_log_key` を設定した組み込みの dummy インプットに対応しています。実際のインプットデータとプラグイン設定に合わせてカラムを調整してください。

2. Docker イメージを実行します。デフォルトでは dummy インプットが 5 秒ごとにテストレコードを送信します。

```bash
docker run --rm -it \
  -e ZEROBUS_ENDPOINT=<workspace-id>.zerobus.<region>.cloud.databricks.com \
  -e ZEROBUS_TABLE_NAME=main.default.zerobus_fluent_bit \
  -e DATABRICKS_HOST=https://<instance>.cloud.databricks.com \
  -e DATABRICKS_CLIENT_ID=<service-principal-client-id> \
  -e DATABRICKS_CLIENT_SECRET=<service-principal-secret> \
  -it fluent-bit-zerobus
```

3. テーブルにレコードが書き込まれていることを確認します。

```sql
SELECT * FROM main.default.zerobus_fluent_bit ORDER BY _time DESC LIMIT 10;
```

## 仕組み

1. Fluent Bit が設定されたインプットからログレコードを収集
2. 本プラグインが各レコードを JSON に変換
3. Fluent Bit チャンク内の全レコードを、公式 Go SDK（gRPC）を使用して ZeroBus に単一のアトミックバッチとして送信
4. ZeroBus が指定された Delta テーブルにデータを直接ストリーミング
5. プラグインはサーバーからの確認応答を待ってから Fluent Bit に完了を通知

## ライセンス

Apache License 2.0
