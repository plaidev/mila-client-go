package mila

import "context"

func call[Response any](ctx context.Context, c *Client, endpoint string, request any, options []RequestOptions) (*Response, error) {
	var result Response
	if err := c.post(ctx, endpoint, request, &result, false, options); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CancelJob(ctx context.Context, request CancelJobRequest, options ...RequestOptions) (*CancelJobResponse, error) {
	return call[CancelJobResponse](ctx, c, "cancelJobFlow", request, options)
}

func (c *Client) CancelQuery(ctx context.Context, request CancelQueryRequest, options ...RequestOptions) (*CancelQueryResponse, error) {
	return call[CancelQueryResponse](ctx, c, "cancelQuery", request, options)
}

func (c *Client) CreateColocationGroup(ctx context.Context, request CreateColocationGroupRequest, options ...RequestOptions) (*CreateColocationGroupResponse, error) {
	return call[CreateColocationGroupResponse](ctx, c, "createColocationGroup", request, options)
}

func (c *Client) CreateTable(ctx context.Context, request CreateTableRequest, options ...RequestOptions) (*CreateTableResponse, error) {
	return call[CreateTableResponse](ctx, c, "createTable", request, options)
}

func (c *Client) DeleteColocationGroup(ctx context.Context, request DeleteColocationGroupRequest, options ...RequestOptions) (*DeleteColocationGroupResponse, error) {
	return call[DeleteColocationGroupResponse](ctx, c, "deleteColocationGroup", request, options)
}

func (c *Client) DeleteTable(ctx context.Context, request DeleteTableRequest, options ...RequestOptions) (*DeleteTableResponse, error) {
	return call[DeleteTableResponse](ctx, c, "deleteTable", request, options)
}

func (c *Client) GetJobStatus(ctx context.Context, request GetJobStatusRequest, options ...RequestOptions) (*GetJobStatusResponse, error) {
	return call[GetJobStatusResponse](ctx, c, "getJobStatus", request, options)
}

func (c *Client) GetRunningSyncJobs(ctx context.Context, request GetRunningJobsRequest, options ...RequestOptions) (*GetRunningJobsResponse, error) {
	return call[GetRunningJobsResponse](ctx, c, "getRunningSyncJobs", request, options)
}

func (c *Client) GetSyncedInterval(ctx context.Context, request GetSyncedIntervalRequest, options ...RequestOptions) (*GetSyncedIntervalResponse, error) {
	return call[GetSyncedIntervalResponse](ctx, c, "getSyncedInterval", request, options)
}

func (c *Client) GetTable(ctx context.Context, request GetTableRequest, options ...RequestOptions) (*GetTableResponse, error) {
	return call[GetTableResponse](ctx, c, "getTable", request, options)
}

func (c *Client) GetView(ctx context.Context, request GetViewRequest, options ...RequestOptions) (*GetViewResponse, error) {
	return call[GetViewResponse](ctx, c, "getView", request, options)
}

func (c *Client) ListTables(ctx context.Context, request ListTablesRequest, options ...RequestOptions) (*ListTablesResponse, error) {
	return call[ListTablesResponse](ctx, c, "listTables", request, options)
}

func (c *Client) RegisterPubSubPublisher(ctx context.Context, request RegisterPubSubPublisherRequest, options ...RequestOptions) (*RegisterPubSubPublisherResponse, error) {
	return call[RegisterPubSubPublisherResponse](ctx, c, "registerPubSubPublisher", request, options)
}

func (c *Client) UnregisterPubSubPublisher(ctx context.Context, request UnregisterPubSubPublisherRequest, options ...RequestOptions) (*UnregisterPubSubPublisherResponse, error) {
	return call[UnregisterPubSubPublisherResponse](ctx, c, "removePubsubPublisher", request, options)
}

func (c *Client) SubscribeBigQueryDataset(ctx context.Context, request SubscribeBigQueryDatasetRequest, options ...RequestOptions) (*SubscribeBigQueryDatasetResponse, error) {
	return call[SubscribeBigQueryDatasetResponse](ctx, c, "subscribeBigQueryDataset", request, options)
}

func (c *Client) TriggerCleanSync(ctx context.Context, request CleanSyncRequest, options ...RequestOptions) (*CleanSyncResponse, error) {
	return call[CleanSyncResponse](ctx, c, "triggerCleanSync", request, options)
}

func (c *Client) TriggerShardKeyMapMerge(ctx context.Context, request TriggerShardKeyMapMergeRequest, options ...RequestOptions) (*TriggerShardKeyMapMergeResponse, error) {
	return call[TriggerShardKeyMapMergeResponse](ctx, c, "triggerShardKeyMapMerge", request, options)
}

func (c *Client) TriggerSync(ctx context.Context, request TriggerSyncRequest, options ...RequestOptions) (*TriggerSyncResponse, error) {
	return call[TriggerSyncResponse](ctx, c, "triggerSync", request, options)
}

func (c *Client) UnsubscribeBigQueryDataset(ctx context.Context, request UnsubscribeBigQueryDatasetRequest, options ...RequestOptions) (*UnsubscribeBigQueryDatasetResponse, error) {
	return call[UnsubscribeBigQueryDatasetResponse](ctx, c, "unsubscribeBigQueryDataset", request, options)
}

func (c *Client) UpdateColocationGroup(ctx context.Context, request UpdateColocationGroupRequest, options ...RequestOptions) (*UpdateColocationGroupResponse, error) {
	return call[UpdateColocationGroupResponse](ctx, c, "updateColocationGroup", request, options)
}

func (c *Client) UpdateTable(ctx context.Context, request UpdateTableRequest, options ...RequestOptions) (*UpdateTableResponse, error) {
	return call[UpdateTableResponse](ctx, c, "updateTable", request, options)
}

// RunSQLRaw は変換前の二次元配列を返す。JSON 数値は json.Number として保持する。
func (c *Client) RunSQLRaw(ctx context.Context, request RunSQLRequest, options ...RequestOptions) (*RawRunSQLResponse, error) {
	var result RawRunSQLResponse
	if err := c.post(ctx, "runSql", request, &result, true, options); err != nil {
		return nil, err
	}
	return &result, nil
}

// RunSQL はカラム名をキーとする map に行を変換する。ARRAY/STRUCT も再帰的に変換する。
func (c *Client) RunSQL(ctx context.Context, request RunSQLRequest, options ...RequestOptions) (*RunSQLResponse, error) {
	raw, err := c.RunSQLRaw(ctx, request, options...)
	if err != nil {
		return nil, err
	}
	schema := raw.Schema
	if schema == nil {
		schema = []ColumnSpec{}
	}
	rows, err := ConvertRunSQLRows(raw.Rows, schema)
	if err != nil {
		return nil, err
	}
	return &RunSQLResponse{QueryID: raw.QueryID, Schema: schema, Rows: rows, Result: raw.Result}, nil
}
