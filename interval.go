package mila

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Interval は開始時刻を含み終了時刻を含まない範囲。JSON では ISO 8601 の start/end。
type Interval struct {
	Start time.Time
	End   time.Time
}

func ParseInterval(value string) (Interval, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return Interval{}, fmt.Errorf("mila: interval must be start/end")
	}
	start, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return Interval{}, fmt.Errorf("mila: invalid interval start: %w", err)
	}
	end, err := time.Parse(time.RFC3339Nano, parts[1])
	if err != nil {
		return Interval{}, fmt.Errorf("mila: invalid interval end: %w", err)
	}
	interval := Interval{Start: start, End: end}
	return interval, interval.validate()
}

func (i Interval) validate() error {
	if i.End.Before(i.Start) {
		return fmt.Errorf("mila: interval end must not precede start")
	}
	return nil
}

func (i Interval) String() string {
	return i.Start.UTC().Format(time.RFC3339Nano) + "/" + i.End.UTC().Format(time.RFC3339Nano)
}

func (i Interval) MarshalJSON() ([]byte, error) {
	if err := i.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(i.String())
}

// UnmarshalJSON は実際の API の文字列形式と OpenAPI のオブジェクト形式を受け取る。
func (i *Interval) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		parsed, err := ParseInterval(value)
		if err == nil {
			*i = parsed
		}
		return err
	}
	var object struct {
		Start       *time.Time `json:"start"`
		End         *time.Time `json:"end"`
		StartMillis *int64     `json:"startMillis"`
		EndMillis   *int64     `json:"endMillis"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	var parsed Interval
	switch {
	case object.Start != nil && object.End != nil:
		parsed = Interval{Start: *object.Start, End: *object.End}
	case object.StartMillis != nil && object.EndMillis != nil:
		parsed = Interval{Start: time.UnixMilli(*object.StartMillis), End: time.UnixMilli(*object.EndMillis)}
	default:
		return fmt.Errorf("mila: interval requires start/end or startMillis/endMillis")
	}
	if err := parsed.validate(); err != nil {
		return err
	}
	*i = parsed
	return nil
}
