package mila

import (
	"encoding/json"
	"reflect"
	"testing"
)

func assertJSON(t *testing.T, value any, want string) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var gotObject, wantObject any
	if err := json.Unmarshal(data, &gotObject); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantObject); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotObject, wantObject) {
		t.Fatalf("JSON = %s, want %s", data, want)
	}
}

func TestOptionalFields(t *testing.T) {
	assertJSON(t, RunSQLRequest{SQL: "SELECT 1"}, `{"sql":"SELECT 1"}`)
	assertJSON(t, RunSQLRequest{SQL: "SELECT 1", AllowOptions: []string{}, QueryableTables: []string{}, AllowMutate: Ptr(false)},
		`{"sql":"SELECT 1","allowOptions":[],"queryableTables":[],"allowMutate":false}`)
	assertJSON(t, UpdateTableRequest{Name: "events", TTLDays: Ptr(int32(0)), CachePolicy: &TableCachePolicySpec{
		Type: "timeseries", PartialReadEnabled: Ptr(false), WarmupColumns: []string{},
	}}, `{"name":"events","ttlDays":0,"cachePolicy":{"type":"timeseries","partialReadEnabled":false,"warmupColumns":[]}}`)
	assertJSON(t, ColumnSpec{Name: "items", Type: ColumnTypeArray, Fields: []ColumnSpec{
		{Type: ColumnTypeStruct, Fields: []ColumnSpec{{Name: "id", Type: ColumnTypeBigInt, Nullable: Ptr(false)}}},
	}}, `{"name":"items","type":"ARRAY","fields":[{"type":"STRUCT","fields":[{"name":"id","type":"BIGINT","nullable":false}]}]}`)
}

func TestDataSources(t *testing.T) {
	assertJSON(t, TriggerSyncRequest{
		TableName: "events", JobID: "sync-1", DataSource: InlineDataSource{Data: []map[string]any{{"id": "one"}}},
		Range: &Range{Start: "a", End: "z"},
	}, `{"tableName":"events","jobId":"sync-1","dataSource":{"type":"inline","data":[{"id":"one"}]},"range":{"start":"a","end":"z"}}`)
	assertJSON(t, BigQueryDataSource{ColumnExtractions: []ColumnExtraction{}, SourceTableID: TableID{Project: "p", Dataset: "d", Table: "t"}, SuffixShardType: SuffixShardTypeNoShard},
		`{"type":"bigQuery","columnExtractions":[],"sourceTableId":{"project":"p","dataset":"d","table":"t"},"suffixShardType":"noShard"}`)
	assertJSON(t, BigQuerySharingDataSource{ColumnExtractions: []ColumnExtraction{}, Listing: "l", TableID: "t", SuffixShardType: SuffixShardTypeDateShard},
		`{"type":"bigQuerySharing","columnExtractions":[],"listing":"l","tableId":"t","suffixShardType":"dateShard"}`)
	assertJSON(t, CleanSyncRequest{TableName: "map", DataSource: InlineShardKeyMapDataSource{Mappings: []KeyMapping{{SourceKey: "a", DestinationKey: "b", Timestamp: Ptr(int64(0))}}}, WarmUp: Ptr(false)},
		`{"tableName":"map","dataSource":{"type":"inlineShardKeyMap","mappings":[{"sourceKey":"a","destinationKey":"b","timestamp":0}]},"warmUp":false}`)
}

func TestColumnExtractions(t *testing.T) {
	for _, test := range []struct {
		value ColumnExtraction
		want  string
	}{
		{PlainColumnExtraction{DestinationColumn: "d", SourceColumn: "s"}, `{"type":"plain","destinationColumn":"d","sourceColumn":"s"}`},
		{StringJSONColumnExtraction{DestinationColumn: "d", JSONColumn: "s", Path: "$.id"}, `{"type":"stringJson","destinationColumn":"d","jsonColumn":"s","path":"$.id"}`},
		{JSONTypeColumnExtraction{DestinationColumn: "d", SourceColumn: "s", Path: "$.id"}, `{"type":"json","destinationColumn":"d","sourceColumn":"s","path":"$.id"}`},
		{JSONObjectColumnExtraction{DestinationColumn: "d", Fields: []JSONObjectField{{Name: "id", SourceColumn: "s"}}}, `{"type":"jsonObject","destinationColumn":"d","fields":[{"name":"id","sourceColumn":"s"}]}`},
		{StructTypeColumnExtraction{DestinationColumn: "d", StructColumnName: "s", Fields: []string{"id"}}, `{"type":"struct","destinationColumn":"d","structColumnName":"s","fields":["id"]}`},
		{ArrayToStr{DestinationColumn: "d", SourceColumn: "s", Delimiter: ""}, `{"type":"arrayToStr","destinationColumn":"d","sourceColumn":"s","delimiter":""}`},
	} {
		t.Run(reflect.TypeOf(test.value).Name(), func(t *testing.T) { assertJSON(t, test.value, test.want) })
	}
}

func TestIntervals(t *testing.T) {
	const value = "2025-03-01T00:00:00Z/2025-03-02T00:00:00Z"
	interval, err := ParseInterval(value)
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, interval, `"`+value+`"`)
	for _, input := range []string{
		`"2025-03-01T09:00:00+09:00/2025-03-02T09:00:00+09:00"`,
		`{"start":"2025-03-01T00:00:00Z","end":"2025-03-02T00:00:00Z"}`,
		`{"startMillis":1740787200000,"endMillis":1740873600000}`,
	} {
		var parsed Interval
		if err := json.Unmarshal([]byte(input), &parsed); err != nil || parsed.String() != value {
			t.Errorf("input = %s, parsed = %s, err = %v", input, parsed, err)
		}
	}
	for _, input := range []string{`"invalid"`, `"bad/2025-03-02T00:00:00Z"`, `"2025-03-01T00:00:00Z/bad"`, `"2025-03-02T00:00:00Z/2025-03-01T00:00:00Z"`, `{}`, `{"startMillis":100,"endMillis":0}`, `null`} {
		var parsed Interval
		if err := json.Unmarshal([]byte(input), &parsed); err == nil {
			t.Errorf("expected error for %s", input)
		}
	}
}

func TestMalformedRows(t *testing.T) {
	for _, test := range []struct {
		value  any
		schema ColumnSpec
	}{
		{map[string]any{}, ColumnSpec{Type: ColumnTypeStruct}},
		{[]any{1}, ColumnSpec{Type: ColumnTypeStruct}},
		{"text", ColumnSpec{Type: ColumnTypeArray, Fields: []ColumnSpec{{Type: ColumnTypeString}}}},
		{[]any{1}, ColumnSpec{Type: ColumnTypeArray}},
		{[]any{[]any{}}, ColumnSpec{Type: ColumnTypeArray, Fields: []ColumnSpec{{Type: ColumnTypeStruct, Fields: []ColumnSpec{{Name: "id", Type: ColumnTypeInt}}}}}},
	} {
		if _, err := ConvertRawCellToObj(test.value, test.schema); err == nil {
			t.Errorf("expected error for %+v", test)
		}
	}
}
