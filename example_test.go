package mila_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	mila "github.com/plaidev/mila-client-go"
)

func ExampleClient_RunSQL() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"queryId":"example","schema":[{"name":"count","type":"BIGINT"}],"rows":[[42]]}`)
	}))
	defer server.Close()
	client, err := mila.New(mila.Config{BaseURL: server.URL})
	if err != nil {
		panic(err)
	}
	result, err := client.RunSQL(context.Background(), mila.RunSQLRequest{SQL: "SELECT count(*) AS count FROM events"})
	if err != nil {
		panic(err)
	}
	count, err := result.Rows[0]["count"].(json.Number).Int64()
	if err != nil {
		panic(err)
	}
	fmt.Println(count)
	// Output: 42
}
