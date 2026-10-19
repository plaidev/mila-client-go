package mila

import "encoding/json"

// ColumnExtraction は BigQuery の列抽出設定。JSON の type は実装が設定する。
type ColumnExtraction interface {
	json.Marshaler
	columnExtraction()
}

type PlainColumnExtraction struct {
	DestinationColumn string `json:"destinationColumn"`
	SourceColumn      string `json:"sourceColumn"`
}

func (PlainColumnExtraction) columnExtraction() {}
func (e PlainColumnExtraction) MarshalJSON() ([]byte, error) {
	type plain PlainColumnExtraction
	return json.Marshal(struct {
		Type string `json:"type"`
		plain
	}{"plain", plain(e)})
}

type StringJSONColumnExtraction struct {
	DestinationColumn string `json:"destinationColumn"`
	JSONColumn        string `json:"jsonColumn"`
	Path              string `json:"path"`
}

func (StringJSONColumnExtraction) columnExtraction() {}
func (e StringJSONColumnExtraction) MarshalJSON() ([]byte, error) {
	type plain StringJSONColumnExtraction
	return json.Marshal(struct {
		Type string `json:"type"`
		plain
	}{"stringJson", plain(e)})
}

type JSONTypeColumnExtraction struct {
	DestinationColumn string `json:"destinationColumn"`
	SourceColumn      string `json:"sourceColumn"`
	Path              string `json:"path"`
}

func (JSONTypeColumnExtraction) columnExtraction() {}
func (e JSONTypeColumnExtraction) MarshalJSON() ([]byte, error) {
	type plain JSONTypeColumnExtraction
	return json.Marshal(struct {
		Type string `json:"type"`
		plain
	}{"json", plain(e)})
}

type JSONObjectColumnExtraction struct {
	DestinationColumn string            `json:"destinationColumn"`
	Fields            []JSONObjectField `json:"fields"`
}

func (JSONObjectColumnExtraction) columnExtraction() {}
func (e JSONObjectColumnExtraction) MarshalJSON() ([]byte, error) {
	type plain JSONObjectColumnExtraction
	return json.Marshal(struct {
		Type string `json:"type"`
		plain
	}{"jsonObject", plain(e)})
}

type StructTypeColumnExtraction struct {
	DestinationColumn string   `json:"destinationColumn"`
	StructColumnName  string   `json:"structColumnName"`
	Fields            []string `json:"fields"`
}

func (StructTypeColumnExtraction) columnExtraction() {}
func (e StructTypeColumnExtraction) MarshalJSON() ([]byte, error) {
	type plain StructTypeColumnExtraction
	return json.Marshal(struct {
		Type string `json:"type"`
		plain
	}{"struct", plain(e)})
}

type ArrayToStr struct {
	DestinationColumn string `json:"destinationColumn"`
	SourceColumn      string `json:"sourceColumn"`
	Delimiter         string `json:"delimiter"`
}

func (ArrayToStr) columnExtraction() {}
func (e ArrayToStr) MarshalJSON() ([]byte, error) {
	type plain ArrayToStr
	return json.Marshal(struct {
		Type string `json:"type"`
		plain
	}{"arrayToStr", plain(e)})
}
