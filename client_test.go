package mila

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := New(Config{BaseURL: server.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestRunSQL(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/mila/v1/runSql" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		for key, want := range map[string]string{
			"Content-Type": "application/json", "Accept": "application/json",
			"Authorization": "Bearer fixed", "Mila-Cluster-Id": "cluster-a",
			"Mila-Principal": "alice,bob", "X-Request-ID": "request-1",
		} {
			if got := r.Header.Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		body, _ := io.ReadAll(r.Body)
		want := `{"sql":"SELECT * FROM events","allowMutate":false,"allowOptions":[],"context":{"timeout":60000},"labels":{"app":"test"},"queryableTables":[]}`
		if string(body) != want {
			t.Errorf("body = %s, want %s", body, want)
		}
		io.WriteString(w, `{"queryId":"q1","schema":[{"name":"id","type":"BIGINT"},{"name":"items","type":"ARRAY","fields":[{"type":"STRUCT","fields":[{"name":"name","type":"STRING"},{"name":"deleted","type":"STRING"}]}]},{"name":"empty","type":"STRING"}],"rows":[[9007199254740993,[["first",null],null],null]]}`)
	})
	client.auth.bearerToken = "fixed"
	headers := http.Header{"X-Request-Id": {"request-1"}}
	result, err := client.ForCluster("cluster-a").RunSQL(context.Background(), RunSQLRequest{
		SQL: "SELECT * FROM events", AllowMutate: Ptr(false), AllowOptions: []string{},
		Context: map[string]any{"timeout": 60000}, Labels: map[string]string{"app": "test"}, QueryableTables: []string{},
	}, RequestOptions{Headers: headers, Principals: []string{"alice", "bob"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []map[string]any{{
		"id": json.Number("9007199254740993"), "items": []any{map[string]any{"name": "first"}, nil}, "empty": nil,
	}}
	if result.QueryID != "q1" || !reflect.DeepEqual(result.Rows, want) {
		t.Fatalf("result = %#v", result)
	}
	if !reflect.DeepEqual(headers, http.Header{"X-Request-Id": {"request-1"}}) {
		t.Fatalf("request changed caller headers: %v", headers)
	}
	if client.clusterID != "" {
		t.Fatal("ForCluster mutated original client")
	}
}

func TestRunSQLDDLAndRaw(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"queryId":"ddl","result":"CREATE TABLE"}`)
	})
	response, err := client.RunSQL(context.Background(), RunSQLRequest{SQL: "CREATE TABLE test (id BIGINT)"})
	if err != nil || response.Result != "CREATE TABLE" || response.Schema == nil || response.Rows == nil || len(response.Rows) != 0 {
		t.Fatalf("response = %+v, err = %v", response, err)
	}
	rawClient := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"queryId":"raw","schema":[{"name":"id","type":"BIGINT"}],"rows":[[9223372036854775807]]}`)
	})
	raw, err := rawClient.RunSQLRaw(context.Background(), RunSQLRequest{SQL: "SELECT id FROM test"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(raw.Rows, [][]any{{json.Number("9223372036854775807")}}) {
		t.Fatalf("raw rows = %#v", raw.Rows)
	}
}

func TestRequestAndAuthenticationErrors(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls atomic.Int32
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
				io.WriteString(w, `{"errorCode":"TABLE_NOT_FOUND"}`)
			})
			client.auth.bearerToken = "fixed"
			_, err := client.GetTable(context.Background(), GetTableRequest{TableName: "missing"}, RequestOptions{MaxRetries: 2})
			var requestErr *RequestError
			if !errors.As(err, &requestErr) || requestErr.StatusCode != status || requestErr.Path != "/mila/v1/getTable" || requestErr.Body != `{"errorCode":"TABLE_NOT_FOUND"}` {
				t.Fatalf("err = %v", err)
			}
			var authErr *AuthenticationError
			if errors.As(err, &authErr) != (status == 401 || status == 403) {
				t.Fatalf("unexpected error type: %T", err)
			}
			if calls.Load() != 1 {
				t.Fatalf("HTTP errors must not cause network retries: %d calls", calls.Load())
			}
		})
	}
}

func TestRefreshAuthenticationOnce(t *testing.T) {
	for _, recover := range []bool{true, false} {
		t.Run(fmt.Sprint(recover), func(t *testing.T) {
			var calls, tokens atomic.Int32
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				attempt := calls.Add(1)
				if got, want := r.Header.Get("Authorization"), fmt.Sprintf("Bearer token-%d", attempt); got != want {
					t.Errorf("Authorization = %q, want %q", got, want)
				}
				if attempt == 1 || !recover {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				io.WriteString(w, `{"name":"events"}`)
			})
			client.auth.provider = func(ctx context.Context) (string, error) {
				return fmt.Sprintf("token-%d", tokens.Add(1)), nil
			}
			response, err := client.GetTable(context.Background(), GetTableRequest{TableName: "events"})
			if recover && (err != nil || response.Name != "events") {
				t.Fatalf("response = %+v, err = %v", response, err)
			}
			var authErr *AuthenticationError
			if !recover && !errors.As(err, &authErr) {
				t.Fatalf("expected AuthenticationError, got %v", err)
			}
			if calls.Load() != 2 || tokens.Load() != 2 {
				t.Fatalf("calls = %d, token fetches = %d", calls.Load(), tokens.Load())
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNetworkRetriesReplayBody(t *testing.T) {
	for _, maxRetries := range []int{0, 1} {
		t.Run(fmt.Sprint(maxRetries), func(t *testing.T) {
			var calls int
			failure := errors.New("connection closed")
			client, err := New(Config{BaseURL: "http://localhost", HTTPClient: &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					body, _ := io.ReadAll(r.Body)
					if string(body) != `{"tableName":"events"}` {
						t.Errorf("attempt %d body = %s", calls, body)
					}
					if calls == 1 {
						return nil, failure
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"name":"events"}`)), Header: make(http.Header)}, nil
				}),
			}})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.GetTable(context.Background(), GetTableRequest{TableName: "events"}, RequestOptions{MaxRetries: maxRetries})
			if maxRetries == 0 {
				var networkErr *NetworkError
				if !errors.Is(err, failure) || !errors.As(err, &networkErr) {
					t.Fatalf("err = %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if calls != maxRetries+1 {
				t.Fatalf("calls = %d", calls)
			}
		})
	}
}

func TestCancellationAndTimeout(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.GetTable(ctx, GetTableRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	_, err = client.GetTable(context.Background(), GetTableRequest{}, RequestOptions{Timeout: 20 * time.Millisecond, MaxRetries: 10})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	client.auth.provider = func(ctx context.Context) (string, error) { <-ctx.Done(); return "", ctx.Err() }
	_, err = client.GetTable(context.Background(), GetTableRequest{}, RequestOptions{Timeout: time.Millisecond})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("token retrieval did not respect deadline: %v", err)
	}
}

func TestDefaultDeadlines(t *testing.T) {
	client, err := New(Config{BaseURL: "http://localhost", HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		want := DefaultTimeout
		if strings.HasSuffix(r.URL.Path, "runSql") {
			want = DefaultQueryTimeout
		}
		if remaining := time.Until(deadline); !ok || remaining > want || remaining < want-time.Second {
			t.Errorf("unexpected deadline for %s: %v", r.URL.Path, remaining)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
	})}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListTables(context.Background(), ListTablesRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.RunSQL(context.Background(), RunSQLRequest{}); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentClusters(t *testing.T) {
	var calls atomic.Int32
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Mila-Cluster-Id") == "" || r.Header.Get("Authorization") != "Bearer shared" {
			t.Errorf("missing cluster/auth headers: %v", r.Header)
		}
		io.WriteString(w, `{"tables":[]}`)
	})
	client.auth.provider = func(context.Context) (string, error) { return "shared", nil }
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := client.ForCluster(fmt.Sprint(i)).ListTables(context.Background(), ListTablesRequest{}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 20 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestInvalidResponses(t *testing.T) {
	for _, body := range []string{"<html>gateway error</html>", `null`, `{} {}`, `{"schema":[{"name":"id","type":"INT"}],"rows":[[]]}`} {
		t.Run(body, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) })
			if _, err := client.RunSQL(context.Background(), RunSQLRequest{}); err == nil {
				t.Fatal("expected error for invalid response")
			}
		})
	}
}

func TestServerValueIntervals(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "getSyncedInterval") {
			io.WriteString(w, `{"interval":["BIGINT",9007199254740993,9007199254740999]}`)
		} else {
			io.WriteString(w, `{"name":"events","partitionSpec":{"type":"fixedList","keyColumn":"id","intervals":[["STRING","a","z"]]}}`)
		}
	})
	interval, err := client.GetSyncedInterval(context.Background(), GetSyncedIntervalRequest{TableName: "events"})
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, interval.Interval, `["BIGINT",9007199254740993,9007199254740999]`)
	if interval.Interval.Start != json.Number("9007199254740993") {
		t.Fatalf("interval boundary lost precision: %v", interval.Interval.Start)
	}
	table, err := client.GetTable(context.Background(), GetTableRequest{TableName: "events"})
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, table.PartitionSpec.Intervals, `[["STRING","a","z"]]`)
}

func TestInvalidConfig(t *testing.T) {
	for _, config := range []Config{
		{}, {BaseURL: "localhost:8090"}, {BaseURL: "ftp://localhost"}, {BaseURL: "http://localhost?x=1"},
		{BaseURL: "http://localhost/?"}, {BaseURL: "http://localhost/#"},
		{BaseURL: "http://user:password@localhost"}, {BaseURL: "http://localhost/#fragment"},
		{BaseURL: "http://localhost", BearerToken: "token", IAPClientID: "iap"},
		{BaseURL: "http://localhost", KeyFilename: "key.json"},
		{BaseURL: "http://localhost", TokenProvider: func(context.Context) (string, error) { return "token", nil }, BearerToken: "token"},
	} {
		if _, err := New(config); err == nil {
			t.Errorf("expected error for config: %+v", config)
		}
	}
}

func TestInvalidOptionsAndEncoding(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected HTTP request") })
	for _, options := range [][]RequestOptions{{{Timeout: -1}}, {{MaxRetries: -1}}, {{}, {}}} {
		if _, err := client.GetTable(context.Background(), GetTableRequest{}, options...); err == nil {
			t.Errorf("expected error for options: %+v", options)
		}
	}
	if _, err := client.RunSQL(context.Background(), RunSQLRequest{Context: map[string]any{"invalid": make(chan int)}}); err == nil {
		t.Fatal("expected JSON encoding error")
	}
}

func TestTokenErrors(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("unauthenticated request") })
	cause := errors.New("token unavailable")
	for _, provider := range []TokenProvider{
		func(context.Context) (string, error) { return "", cause },
		func(context.Context) (string, error) { return "", nil },
	} {
		client.auth.provider = provider
		_, err := client.ListTables(context.Background(), ListTablesRequest{})
		var authErr *AuthenticationError
		if !errors.As(err, &authErr) {
			t.Fatalf("err = %v", err)
		}
	}
	client.auth.provider = nil
	client.auth.iapClientID = "iap"
	client.auth.keyFilename = t.TempDir() + "/missing-key.json"
	_, err := client.ListTables(context.Background(), ListTablesRequest{})
	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Fatalf("err = %v", err)
	}
}
