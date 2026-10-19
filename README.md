# Mila Go client

TypeScript の `@plaidev/mila-client` に対応する Go ライブラリ。Go 1.25 以上が必要です。

SQL 実行、テーブル・ビュー参照、テーブル・colocation group 管理、同期ジョブ管理、BigQuery dataset 購読、Pub/Sub publisher 管理を提供します。API のパスとリクエスト・レスポンスのフィールドは、`testdata/openapi.yaml` の SDK 向けスキーマに対応しています。

## 導入

独立した Go module です。

```text
github.com/plaidev/mila-client-go
```

公開バージョンは GitHub の public repository から取得できます。利用側のトークンや独自の `GOPROXY` 設定は不要です。

```sh
go get github.com/plaidev/mila-client-go@v0.1.0
```

ローカルで利用する場合は、利用先の `go.mod` に次を追加してください。`replace` のパスは checkout の場所に合わせます。

```go
require github.com/plaidev/mila-client-go v0.0.0

replace github.com/plaidev/mila-client-go => /path/to/mila/mila-client-go
```

この repository は配布用のミラーです。開発元の Mila モノレポジトリで `mila-client-go/v0.1.0` のようなタグを付けると、CI がクライアントだけを同期し、この repository に `v0.1.0` タグを作成します。既存の TypeScript client とは独立してバージョンを管理します。

## SQL を実行する

```go
package main

import (
    "context"
    "fmt"
    "log"

    mila "github.com/plaidev/mila-client-go"
)

func main() {
    client, err := mila.New(mila.Config{
        BaseURL: "http://localhost:8090",
    })
    if err != nil {
        log.Fatal(err)
    }

    result, err := client.RunSQL(context.Background(), mila.RunSQLRequest{
        SQL: "SELECT * FROM events LIMIT 10",
    })
    if err != nil {
        log.Fatal(err)
    }
    for _, row := range result.Rows {
        fmt.Println(row)
    }
}
```

`BaseURL` は `/mila` を含まないサーバーの URL です。プロキシのパス接頭辞がある場合は `https://example.com/proxy` のように指定できます。

`RunSQL` の行は `[]map[string]any` です。STRUCT は map に、ARRAY はスライスに再帰的に変換します。トップレベルの null は保持し、STRUCT 内の null フィールドは TypeScript 版と同様に省略します。DDL のメッセージは `Result` に入ります。変換前の二次元配列が必要な場合は `RunSQLRaw` を使ってください。

SQL 結果の JSON 数値は、BIGINT の精度を保つため `json.Number` として返します。必要な型への変換は呼び出し側で明示します。

```go
number := result.Rows[0]["count"].(json.Number)
count, err := number.Int64()
```

## 認証と複数クラスタ

```go
client, err := mila.New(mila.Config{
    BaseURL:     "https://mila.example.com",
    IAPClientID: "your-iap-client-id",
    // サービスアカウント鍵を使う場合に指定する。
    // KeyFilename: "/path/to/service-account.json",
})
if err != nil {
    return err
}

tables, err := client.ForCluster("cluster-id").ListTables(ctx, mila.ListTablesRequest{
    PageSize: mila.Ptr(int32(100)),
})
```

IAP は Google 公式の `cloud.google.com/go/auth/credentials/idtoken` を使い、Application Default Credentials または指定したサービスアカウント鍵で ID token を取得します。ADC は ID token を発行できるサービスアカウント・その偽装・対応する実行環境などが必要です。

固定 token は `BearerToken`、独自の認証処理は `TokenProvider func(context.Context) (string, error)` で指定できます。`IAPClientID`、`BearerToken`、`TokenProvider` は同時に指定できません。認証指定を省略すると Authorization ヘッダーを付けません。IAP と TokenProvider は HTTP 401/403 時に一度だけ認証を取得し直して再送します。

`Client` と `ForCluster` の戻り値は複数の goroutine から共有できます。独自の TokenProvider も並行呼び出しに対応させてください。`ForCluster` は元の client を変更せず、HTTP 接続と認証情報を共有します。

## リクエスト設定

各メソッドの末尾には、任意で一つの `RequestOptions` を指定できます。

```go
result, err := client.RunSQL(ctx, mila.RunSQLRequest{
    SQL: "SELECT * FROM events LIMIT 10",
}, mila.RequestOptions{
    Timeout:    30 * time.Second,
    Principals: []string{"alice", "bob"},
    Headers:    http.Header{"X-Request-Id": {"request-123"}},
})
```

通常の API は 10 秒、SQL は 70 秒がデフォルトです。`context` の期限が先ならそちらを使います。タイムアウトは認証取得と再試行も含みます。`HTTPClient` を差し替える場合、その `Timeout` も適用されます。`Principals` は SQL リクエストの `Mila-Principal` ヘッダーに設定します。

通信失敗時の再試行はデフォルトで無効です。追加試行回数を `MaxRetries` に指定できます。更新処理はサーバー側で既に実行されている可能性があるため、再送可能な処理だけで指定してください。同期ジョブは安定した `JobID` を指定して重複実行を防げます。HTTP 4xx/5xx に対する通信再試行は行いません。

`Config.Logger` に `*slog.Logger` を指定すると、API のパスと HTTP status を debug レベルで記録します。

## テーブルと同期

```go
_, err := client.CreateTable(ctx, mila.CreateTableRequest{
    Name:           "events",
    Type:           mila.TableTypeColumnar,
    ShardKeyColumn: "timestamp",
    PartitionSpec: &mila.PartitionSpec{
        Type:      mila.PartitionTypeTimeseries,
        KeyColumn: "timestamp",
    },
    SortKeys: []mila.SortKeySpec{
        {Type: mila.SortKeyTypePlain, Column: "timestamp"},
    },
    Schema: []mila.ColumnSpec{
        {Name: "timestamp", Type: mila.ColumnTypeBigInt},
        {Name: "name", Type: mila.ColumnTypeString},
    },
})
if err != nil {
    return err
}

job, err := client.TriggerSync(ctx, mila.TriggerSyncRequest{
    TableName: "events",
    JobID:     "my-app:batch-42",
    DataSource: mila.InlineDataSource{
        Data: []map[string]any{
            {"timestamp": int64(1704067200000), "name": "example"},
        },
    },
})
```

`DataSource` は `BigQueryDataSource`、`BigQuerySharingDataSource`、`InlineDataSource`、`InlineShardKeyMapDataSource` を指定できます。JSON の discriminator `type` は自動で付与します。`TriggerCleanSync` は `InlineDataSource` を受け付けません。

任意の数値・真偽値はポインタ型です。`mila.Ptr(false)` や `mila.Ptr(int32(0))` で省略とゼロ値を区別します。任意のスライスは `nil` なら省略し、空の非 nil スライスなら `[]` を送ります。例えば `RunSQLRequest.AllowOptions` は `nil` が制限なし、`[]string{}` が SET オプションを許可しない指定になります。

旧同期 API 向けの `Interval` は `ParseInterval("2025-03-01T00:00:00Z/2025-03-02T00:00:00Z")` で作成できます。JSON では `start/end` 文字列にし、文字列と `start`/`end` または `startMillis`/`endMillis` を持つオブジェクトを読み込めます。新しい同期処理では `Range` を優先してください。

`GetSyncedIntervalResponse.Interval` と `PartitionSpec.Intervals` は汎用範囲の `ValueInterval` です。Java サーバーの `common.Interval` に合わせて `["BIGINT", start, end]` のような配列で送受信し、開始・終了の両方を含みます。境界値の数値も `json.Number` で読み込みます。OpenAPI ではこの型が時刻の Interval と混同されているため、実際のサーバーの形式に合わせています。

## エラー

`errors.As` で `*mila.RequestError`（Path、StatusCode、Body）、`*mila.AuthenticationError`、`*mila.NetworkError` を判定できます。HTTP 401/403 の AuthenticationError は RequestError を内包します。中断・期限超過は `errors.Is(err, context.Canceled)` / `errors.Is(err, context.DeadlineExceeded)` で判定できます。

## 開発

```sh
cd mila-client-go
gofmt -w .
go test -race ./...
go vet ./...
```

API 型・メソッドは手書きで管理しています。開発元では TypeScript client と共通の SDK フィルターから `testdata/openapi.yaml` を生成し、元仕様との一致を CI で検証しています。同期・取り込み JobFlow の停止は TypeScript 版と同じ `CancelJob` メソッドで `/cancelJobFlow` を呼びます。Query Job API は SDK の対応範囲に含みません。

`TestOpenAPIContract` がすべての SDK 対象エンドポイントをローカル HTTP サーバーで呼び出し、パスとリクエスト・レスポンスのフィールドが共通仕様に対応することを検証します。スナップショットを同梱しているので、公開 repository だけで全テストを実行できます。実環境の IAP 認証・Mila サーバーへの接続には別途検証が必要です。

## ライセンス

MIT License。詳細は [LICENSE](LICENSE) を参照してください。
