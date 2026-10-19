package mila

import (
	"context"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// 共通仕様との乖離を検出する。公開メソッドを実際に HTTP 経由で呼ぶ。
func TestOpenAPIContract(t *testing.T) {
	data, err := os.ReadFile("testdata/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	type schema struct {
		Ref        string                    `yaml:"$ref"`
		Properties map[string]map[string]any `yaml:"properties"`
	}
	type media struct {
		Schema schema `yaml:"schema"`
	}
	type content struct {
		Content map[string]media `yaml:"content"`
	}
	var spec struct {
		Paths map[string]struct {
			Post struct {
				OperationID string             `yaml:"operationId"`
				RequestBody content            `yaml:"requestBody"`
				Responses   map[string]content `yaml:"responses"`
			} `yaml:"post"`
		} `yaml:"paths"`
		Components struct {
			Schemas map[string]schema `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	checkFields := func(t *testing.T, typ reflect.Type, ref string) {
		t.Helper()
		name := strings.TrimPrefix(ref, "#/components/schemas/")
		model, ok := spec.Components.Schemas[name]
		if !ok {
			t.Fatalf("unknown schema: %s", ref)
		}
		fields := make(map[string]bool)
		for index := range typ.NumField() {
			fields[strings.Split(typ.Field(index).Tag.Get("json"), ",")[0]] = true
		}
		for property := range model.Properties {
			if !fields[property] {
				t.Errorf("%s is missing OpenAPI property %s", typ, property)
			}
		}
	}
	for path, item := range spec.Paths {
		t.Run(item.Post.OperationID, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/mila"+path {
					t.Errorf("request = %s %s, want POST /mila%s", r.Method, r.URL.Path, path)
				}
				io.WriteString(w, `{}`)
			})
			methodName := strings.ToUpper(item.Post.OperationID[:1]) + item.Post.OperationID[1:]
			// 利用者向けの名前は TypeScript client に合わせる。
			switch methodName {
			case "CancelJobFlow":
				methodName = "CancelJob"
			case "RunSql":
				methodName = "RunSQL"
			case "SyncedInterval":
				methodName = "GetSyncedInterval"
			case "CleanSync":
				methodName = "TriggerCleanSync"
			case "GetRunningJobs":
				methodName = "GetRunningSyncJobs"
			}
			method := reflect.ValueOf(client).MethodByName(methodName)
			if !method.IsValid() {
				t.Fatalf("missing client method: %s", methodName)
			}
			checkFields(t, method.Type().In(1), item.Post.RequestBody.Content["application/json"].Schema.Ref)
			response := item.Post.Responses["default"]
			if _, ok := item.Post.Responses["200"]; ok {
				response = item.Post.Responses["200"]
			}
			checkFields(t, method.Type().Out(0).Elem(), response.Content["application/json"].Schema.Ref)
			result := method.CallSlice([]reflect.Value{
				reflect.ValueOf(context.Background()), reflect.Zero(method.Type().In(1)), reflect.ValueOf([]RequestOptions{}),
			})
			if !result[1].IsNil() {
				t.Fatal(result[1].Interface())
			}
		})
	}
}
