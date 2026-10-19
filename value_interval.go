package mila

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ValueInterval は開始・終了を含む汎用の範囲。
// 同期済み範囲と fixedList partition は、サーバーの common.Interval に対応する
// [type, start, end] 形式を使う。数値の境界値は json.Number として読み込む。
type ValueInterval struct {
	Type  ColumnType
	Start any
	End   any
}

func (i ValueInterval) MarshalJSON() ([]byte, error) {
	if i.Type == "" || i.Start == nil || i.End == nil {
		return nil, fmt.Errorf("mila: value interval requires type, start and end")
	}
	return json.Marshal([3]any{i.Type, i.Start, i.End})
}

func (i *ValueInterval) UnmarshalJSON(data []byte) error {
	var values []json.RawMessage
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	if len(values) != 3 {
		return fmt.Errorf("mila: value interval must be [type, start, end]")
	}
	var parsed ValueInterval
	if err := json.Unmarshal(values[0], &parsed.Type); err != nil {
		return err
	}
	for index, target := range []*any{&parsed.Start, &parsed.End} {
		decoder := json.NewDecoder(bytes.NewReader(values[index+1]))
		decoder.UseNumber()
		if err := decoder.Decode(target); err != nil {
			return err
		}
		switch (*target).(type) {
		case string, json.Number, bool:
		default:
			return fmt.Errorf("mila: value interval boundary must be a scalar")
		}
	}
	if parsed.Type == "" {
		return fmt.Errorf("mila: value interval requires a type")
	}
	*i = parsed
	return nil
}
