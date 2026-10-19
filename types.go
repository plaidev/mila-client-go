package mila

// API 型は testdata/openapi.yaml の SDK 向けスキーマに対応する。
// 任意の数値・真偽値はポインタで省略とゼロ値を区別する。
// 任意のスライスは omitzero で nil のみ省略し、空配列はそのまま送信する。

// Ptr は任意フィールドに指定する値のポインタを返す。
func Ptr[T any](value T) *T { return &value }

type TableType string

const (
	TableTypeColumnar    TableType = "columnar"
	TableTypeAttribute   TableType = "attribute"
	TableTypeShardKeyMap TableType = "shardKeyMap"
)

type ColumnType string

const (
	ColumnTypeDouble    ColumnType = "DOUBLE"
	ColumnTypeFloat     ColumnType = "FLOAT"
	ColumnTypeBigInt    ColumnType = "BIGINT"
	ColumnTypeString    ColumnType = "STRING"
	ColumnTypeArray     ColumnType = "ARRAY"
	ColumnTypeStruct    ColumnType = "STRUCT"
	ColumnTypeBoolean   ColumnType = "BOOLEAN"
	ColumnTypeInt       ColumnType = "INT"
	ColumnTypeDynamic   ColumnType = "DYNAMIC"
	ColumnTypeJSON      ColumnType = "JSON"
	ColumnTypeTimestamp ColumnType = "TIMESTAMP"
	ColumnTypeComplex   ColumnType = "COMPLEX"
	ColumnTypeKLLSketch ColumnType = "KLL_SKETCH"
	ColumnTypeHLLSketch ColumnType = "HLL_SKETCH"
)

type StringEncoding string

const (
	StringEncodingGlobalDict StringEncoding = "GLOBAL_DICT"
	StringEncodingLocalDict  StringEncoding = "LOCAL_DICT"
	StringEncodingNone       StringEncoding = "NONE"
)

// ColumnSpec の ARRAY の Fields は name を持たない要素定義一つを指定する。
// STRUCT の Fields は名前付きフィールドの一覧を指定する。
type ColumnSpec struct {
	Name              string         `json:"name,omitempty"`
	Type              ColumnType     `json:"type"`
	Fields            []ColumnSpec   `json:"fields,omitzero"`
	Nullable          *bool          `json:"nullable,omitempty"`
	Description       string         `json:"description,omitempty"`
	DictionaryEncoded *bool          `json:"dictionaryEncoded,omitempty"` // Deprecated: StringEncoding を使う。
	StringEncoding    StringEncoding `json:"stringEncoding,omitempty"`
	FixedArrayLength  *int32         `json:"fixedArrayLength,omitempty"`
}

type PartitionType string

const (
	PartitionTypeTimeseries  PartitionType = "timeseries"
	PartitionTypeStandard    PartitionType = "standard"
	PartitionTypeAttribute   PartitionType = "attribute"
	PartitionTypeFixedList   PartitionType = "fixedList"
	PartitionTypeShardKeyMap PartitionType = "shardKeyMap"
)

// PartitionSpec は Type に対応する設定を指定する。
// timeseries/standard/attribute/fixedList では KeyColumn が必要。
// fixedList は Intervals、attribute は任意の InitialShard を使う。
type PartitionSpec struct {
	Type         PartitionType   `json:"type"`
	KeyColumn    string          `json:"keyColumn,omitempty"`
	InitialShard *int32          `json:"initialShard,omitempty"`
	Intervals    []ValueInterval `json:"intervals,omitzero"`
}

type SortKeyType string

const (
	SortKeyTypePlain      SortKeyType = "plain"
	SortKeyTypeTimeBucket SortKeyType = "timeBucket"
)

type SortKeySpec struct {
	Type        SortKeyType `json:"type"`
	Column      string      `json:"column"`
	Granularity string      `json:"granularity,omitempty"` // timeBucket のみ。例: HOUR, DAY。
}

type ReplicationStrategy string

const (
	ReplicationStrategyEven      ReplicationStrategy = "even"
	ReplicationStrategyBroadcast ReplicationStrategy = "broadcast"
)

type ColocationGroupConfigWithName struct {
	Name                string              `json:"name,omitempty"`
	AnchorTable         string              `json:"anchorTable,omitempty"`
	ReplicationNum      *int32              `json:"replicationNum,omitempty"`
	ReplicationStrategy ReplicationStrategy `json:"replicationStrategy,omitempty"`
}

type MilaTableTuningConfig struct {
	IdealRowNumPerShard *int32 `json:"idealRowNumPerShard,omitempty"`
	MinShardNum         *int32 `json:"minShardNum,omitempty"`
	ReplicationNum      *int32 `json:"replicationNum,omitempty"`
}

type TableCachePolicySpec struct {
	Type                    string   `json:"type"` // timeseries
	PartialReadEnabled      *bool    `json:"partialReadEnabled,omitempty"`
	TimeseriesFullReadHours *int32   `json:"timeseriesFullReadHours,omitempty"`
	WarmupColumnPaths       []string `json:"warmupColumnPaths,omitzero"`
	WarmupColumns           []string `json:"warmupColumns,omitzero"`
}

type KeyMappabilityRule struct {
	Type     string   `json:"type"` // mappablePrefixKeys
	Prefixes []string `json:"prefixes,omitzero"`
}

type ShardKeyMapConfig struct {
	MappingRule *KeyMappabilityRule `json:"mappingRule,omitempty"`
}

// IndexSpec は ann_jv_1 インデックスの設定を保持する。
type IndexSpec struct {
	Type             string   `json:"type"`
	Column           string   `json:"column,omitempty"`
	Alpha            *float32 `json:"alpha,omitempty"`
	BeamWidth        *int32   `json:"beamWidth,omitempty"`
	ClusterCount     *int32   `json:"clusterCount,omitempty"`
	Dimension        *int32   `json:"dimension,omitempty"`
	EfConstruction   *int32   `json:"efConstruction,omitempty"`
	M                *int32   `json:"m,omitempty"`
	NeighborOverflow *float32 `json:"neighborOverflow,omitempty"`
	Quantize         *bool    `json:"quantize,omitempty"`
	Similarity       string   `json:"similarity,omitempty"`
}

type JSONObjectField struct {
	Name         string            `json:"name,omitempty"`
	SourceColumn string            `json:"sourceColumn,omitempty"`
	Fields       []JSONObjectField `json:"fields,omitzero"`
}

type TableID struct {
	Project         string   `json:"project,omitempty"`
	Dataset         string   `json:"dataset,omitempty"`
	Table           string   `json:"table,omitempty"`
	IAMResourceName string   `json:"iamresourceName,omitempty"`
	ProjectID       *TableID `json:"projectId,omitempty"`
}

type KeyMapping struct {
	SourceKey      string `json:"sourceKey,omitempty"`
	DestinationKey string `json:"destinationKey,omitempty"`
	Timestamp      *int64 `json:"timestamp,omitempty"`
}

type SyncOptions struct {
	RemoveCyclicMappings     *bool `json:"removeCyclicMappings,omitempty"`
	UseSegmentationOptimizer *bool `json:"useSegmentationOptimizer,omitempty"`
}

// Range の Start/End には shardKey に対応する数値または文字列を指定する。
// 数値の変換は行わない。
type Range struct {
	Start any `json:"start"`
	End   any `json:"end"`
}

type SyncPriority string

const (
	SyncPriorityHigh SyncPriority = "high"
	SyncPriorityMid  SyncPriority = "mid"
	SyncPriorityLow  SyncPriority = "low"
)

type CreateTableRequest struct {
	Name                string                         `json:"name"`
	Type                TableType                      `json:"type"`
	ShardKeyColumn      string                         `json:"shardKeyColumn"`
	SortKeys            []SortKeySpec                  `json:"sortKeys"`
	PartitionSpec       *PartitionSpec                 `json:"partitionSpec,omitempty"`
	Schema              []ColumnSpec                   `json:"schema,omitzero"`
	Description         string                         `json:"description,omitempty"`
	ColocationGroup     *ColocationGroupConfigWithName `json:"colocationGroup,omitempty"`
	ColocationGroupName string                         `json:"colocationGroupName,omitempty"`
	IndexSpecs          []IndexSpec                    `json:"indexSpecs,omitzero"`
	PrimaryKeys         []string                       `json:"primaryKeys,omitzero"`
	ShardCount          *int32                         `json:"shardCount,omitempty"`
	ShardKeyMapConfig   *ShardKeyMapConfig             `json:"shardKeyMapConfig,omitempty"`
	TTLDays             *int32                         `json:"ttlDays,omitempty"`
	TuningConfig        *MilaTableTuningConfig         `json:"tuningConfig,omitempty"`
	CachePolicy         *TableCachePolicySpec          `json:"cachePolicy,omitempty"`
}

type GetTableRequest struct {
	TableName string `json:"tableName"`
}

type GetTableResponse struct {
	Name              string                         `json:"name,omitempty"`
	TableID           string                         `json:"tableId,omitempty"`
	TableType         TableType                      `json:"tableType,omitempty"`
	ShardKeyColumn    string                         `json:"shardKeyColumn,omitempty"`
	Schema            []ColumnSpec                   `json:"schema,omitzero"`
	Description       string                         `json:"description,omitempty"`
	PartitionSpec     *PartitionSpec                 `json:"partitionSpec,omitempty"`
	SortKeys          []SortKeySpec                  `json:"sortKeys,omitzero"`
	PrimaryKeys       []string                       `json:"primaryKeys,omitzero"`
	ColocationGroup   *ColocationGroupConfigWithName `json:"colocationGroup,omitempty"`
	ShardKeyMapConfig *ShardKeyMapConfig             `json:"shardKeyMapConfig,omitempty"`
	TTLDays           *int32                         `json:"ttlDays,omitempty"`
	TuningConfig      *MilaTableTuningConfig         `json:"tuningConfig,omitempty"`
	CachePolicy       *TableCachePolicySpec          `json:"cachePolicy,omitempty"`
}

type UpdateTableRequest struct {
	Name        string                `json:"name"`
	ShardCount  *int32                `json:"shardCount,omitempty"`
	TTLDays     *int32                `json:"ttlDays,omitempty"`
	CachePolicy *TableCachePolicySpec `json:"cachePolicy,omitempty"`
}

type DeleteTableRequest struct {
	TableName string `json:"tableName"`
}

type GetViewRequest struct {
	ViewName string `json:"viewName"`
}

type GetViewResponse struct {
	Name            string   `json:"name,omitempty"`
	ViewID          string   `json:"viewId,omitempty"`
	SQLText         string   `json:"sqlText,omitempty"`
	Description     string   `json:"description,omitempty"`
	ColumnAliases   []string `json:"columnAliases,omitzero"`
	CreatedAtMillis *int64   `json:"createdAtMillis,omitempty"`
}

type ListTablesRequest struct {
	PageSize *int32 `json:"pageSize,omitempty"`
}

type ListTablesResponseRecord struct {
	Name string `json:"name,omitempty"`
}

type ListTablesResponse struct {
	Tables []ListTablesResponseRecord `json:"tables,omitzero"`
}

type CreateColocationGroupRequest struct {
	ColocationGroupName string              `json:"colocationGroupName"`
	AnchorTable         string              `json:"anchorTable,omitempty"`
	ReplicationNum      *int32              `json:"replicationNum,omitempty"`
	ReplicationStrategy ReplicationStrategy `json:"replicationStrategy,omitempty"`
}

type UpdateColocationGroupRequest = CreateColocationGroupRequest

type DeleteColocationGroupRequest struct {
	ColocationGroupName string `json:"colocationGroupName"`
	IgnoreMissing       *bool  `json:"ignoreMissing,omitempty"`
}

type GetRunningJobsRequest struct {
	TableName string `json:"tableName"`
}

type GetRunningJobsResponse struct {
	JobIDs []string `json:"jobIds,omitzero"`
}

type GetSyncedIntervalRequest struct {
	TableName string `json:"tableName"`
}

type GetSyncedIntervalResponse struct {
	Interval *ValueInterval `json:"interval,omitempty"`
}

type TriggerSyncRequest struct {
	TableName  string       `json:"tableName"`
	DataSource DataSource   `json:"dataSource"`
	Range      *Range       `json:"range,omitempty"`
	JobID      string       `json:"jobId,omitempty"`
	Priority   SyncPriority `json:"priority,omitempty"`
	Options    *SyncOptions `json:"options,omitempty"`
	Interval   *Interval    `json:"interval,omitempty"` // Deprecated: Range を使う。
}

type CleanSyncRequest struct {
	TableName  string              `json:"tableName"`
	DataSource CleanSyncDataSource `json:"dataSource"`
	JobID      string              `json:"jobId,omitempty"`
	Options    *SyncOptions        `json:"options,omitempty"`
	WarmUp     *bool               `json:"warmUp,omitempty"`
	Interval   *Interval           `json:"interval,omitempty"`
}

type TriggerShardKeyMapMergeRequest struct {
	TableName            string `json:"tableName"`
	ShardKeyMapTableName string `json:"shardKeyMapTableName"`
	JobID                string `json:"jobId,omitempty"`
	MapLookbackMillis    *int64 `json:"mapLookbackMillis,omitempty"`
}

type GetJobStatusRequest struct {
	JobID string `json:"jobId"`
}

type GetJobStatusResponse struct {
	Status string `json:"status,omitempty"`
}

type CancelJobRequest struct {
	JobID string `json:"jobId"`
}

type CancelJobResponse struct {
	Message string `json:"message,omitempty"`
	Success bool   `json:"success"`
}

type CancelQueryRequest struct {
	QueryID string `json:"queryId"`
}

type SubscribeBigQueryDatasetRequest struct {
	ListingName string `json:"listingName"`
}

type UnsubscribeBigQueryDatasetRequest = SubscribeBigQueryDatasetRequest

type RegisterPubSubPublisherRequest struct {
	ServiceAccountEmail string `json:"serviceAccountEmail"`
}

type UnregisterPubSubPublisherRequest = RegisterPubSubPublisherRequest

type RunSQLRequest struct {
	SQL             string            `json:"sql"`
	AllowMutate     *bool             `json:"allowMutate,omitempty"`
	AllowOptions    []string          `json:"allowOptions,omitzero"`
	Context         map[string]any    `json:"context,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	QueryableTables []string          `json:"queryableTables,omitzero"`
}

type RawRunSQLResponse struct {
	QueryID string       `json:"queryId"`
	Schema  []ColumnSpec `json:"schema"`
	Rows    [][]any      `json:"rows"`
	Result  string       `json:"result,omitempty"`
}

type RunSQLResponse struct {
	QueryID string           `json:"queryId"`
	Schema  []ColumnSpec     `json:"schema"`
	Rows    []map[string]any `json:"rows"`
	Result  string           `json:"result,omitempty"`
}

type ResultResponse struct {
	Result string `json:"result,omitempty"`
}

type JobResponse struct {
	JobID string `json:"jobId,omitempty"`
}

type CreateTableResponse = ResultResponse
type UpdateTableResponse = ResultResponse
type DeleteTableResponse = ResultResponse
type CreateColocationGroupResponse = ResultResponse
type UpdateColocationGroupResponse = ResultResponse
type DeleteColocationGroupResponse = ResultResponse
type CancelQueryResponse = ResultResponse
type SubscribeBigQueryDatasetResponse = ResultResponse
type UnsubscribeBigQueryDatasetResponse = ResultResponse
type RegisterPubSubPublisherResponse = ResultResponse
type UnregisterPubSubPublisherResponse = ResultResponse
type TriggerSyncResponse = JobResponse
type CleanSyncResponse = JobResponse
type TriggerShardKeyMapMergeResponse = JobResponse
