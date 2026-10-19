package mila

import "encoding/json"

// DataSource は同期元を表す。各実装が JSON の type を自動で設定する。
type DataSource interface {
	json.Marshaler
	dataSource()
}

// CleanSyncDataSource は全件同期に使えるデータソース。InlineDataSource は使えない。
type CleanSyncDataSource interface {
	DataSource
	cleanSyncDataSource()
}

type SuffixShardType string

const (
	SuffixShardTypeNoShard   SuffixShardType = "noShard"
	SuffixShardTypeDateShard SuffixShardType = "dateShard"
)

type BigQueryDataSource struct {
	ColumnExtractions []ColumnExtraction `json:"columnExtractions"`
	SourceTableID     TableID            `json:"sourceTableId"`
	SuffixShardType   SuffixShardType    `json:"suffixShardType"`
}

func (BigQueryDataSource) dataSource()          {}
func (BigQueryDataSource) cleanSyncDataSource() {}
func (d BigQueryDataSource) MarshalJSON() ([]byte, error) {
	type plain BigQueryDataSource
	return json.Marshal(struct {
		Type string `json:"type"`
		plain
	}{"bigQuery", plain(d)})
}

type BigQuerySharingDataSource struct {
	ColumnExtractions []ColumnExtraction `json:"columnExtractions"`
	Listing           string             `json:"listing"`
	TableID           string             `json:"tableId"`
	SuffixShardType   SuffixShardType    `json:"suffixShardType"`
}

func (BigQuerySharingDataSource) dataSource()          {}
func (BigQuerySharingDataSource) cleanSyncDataSource() {}
func (d BigQuerySharingDataSource) MarshalJSON() ([]byte, error) {
	type plain BigQuerySharingDataSource
	return json.Marshal(struct {
		Type string `json:"type"`
		plain
	}{"bigQuerySharing", plain(d)})
}

type InlineDataSource struct {
	Data []map[string]any `json:"data"`
}

func (InlineDataSource) dataSource() {}
func (d InlineDataSource) MarshalJSON() ([]byte, error) {
	type plain InlineDataSource
	return json.Marshal(struct {
		Type string `json:"type"`
		plain
	}{"inline", plain(d)})
}

type InlineShardKeyMapDataSource struct {
	Mappings []KeyMapping `json:"mappings"`
}

func (InlineShardKeyMapDataSource) dataSource()          {}
func (InlineShardKeyMapDataSource) cleanSyncDataSource() {}
func (d InlineShardKeyMapDataSource) MarshalJSON() ([]byte, error) {
	type plain InlineShardKeyMapDataSource
	return json.Marshal(struct {
		Type string `json:"type"`
		plain
	}{"inlineShardKeyMap", plain(d)})
}
