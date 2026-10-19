package mila

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultTimeout      = 10 * time.Second
	DefaultQueryTimeout = 70 * time.Second
)

// Config は接続先と認証方法を指定する。
// BearerToken、TokenProvider、IAPClientID はいずれか一つだけ指定できる。
// KeyFilename は IAPClientID と組み合わせてサービスアカウント鍵を指定する。
type Config struct {
	BaseURL       string
	BearerToken   string
	TokenProvider TokenProvider
	IAPClientID   string
	KeyFilename   string
	HTTPClient    *http.Client
	Logger        *slog.Logger
}

// RequestOptions は呼び出しごとの設定を指定する。省略時はデフォルト設定を使う。
type RequestOptions struct {
	Headers    http.Header
	Timeout    time.Duration
	MaxRetries int // 通信失敗時の追加試行回数。デフォルトは 0。
	Principals []string
}

// Client は Mila API クライアント。New で生成する。
type Client struct {
	baseURL    string
	httpClient *http.Client
	auth       *authentication
	logger     *slog.Logger
	clusterID  string
}

// New はクライアントを生成する。IAP の認証情報は初回の API 呼び出しで取得する。
func New(config Config) (*Client, error) {
	u, err := url.Parse(config.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || strings.ContainsAny(config.BaseURL, "?#") || u.User != nil {
		return nil, fmt.Errorf("mila: BaseURL must be an absolute HTTP(S) URL without credentials, query or fragment")
	}
	methods := 0
	for _, enabled := range []bool{config.BearerToken != "", config.TokenProvider != nil, config.IAPClientID != ""} {
		if enabled {
			methods++
		}
	}
	if methods > 1 {
		return nil, fmt.Errorf("mila: BearerToken, TokenProvider and IAPClientID are mutually exclusive")
	}
	if config.KeyFilename != "" && config.IAPClientID == "" {
		return nil, fmt.Errorf("mila: KeyFilename requires IAPClientID")
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		baseURL: strings.TrimRight(config.BaseURL, "/"), httpClient: httpClient,
		auth: &authentication{
			bearerToken: config.BearerToken, provider: config.TokenProvider,
			iapClientID: config.IAPClientID, keyFilename: config.KeyFilename,
		},
		logger: config.Logger,
	}, nil
}

// ForCluster は指定クラスタ向けのクライアントを返す。認証と HTTP 接続を共有する。
func (c *Client) ForCluster(clusterID string) *Client {
	clone := *c
	clone.clusterID = clusterID
	return &clone
}

func (c *Client) post(ctx context.Context, endpoint string, payload, result any, query bool, options []RequestOptions) error {
	if len(options) > 1 {
		return fmt.Errorf("mila: at most one RequestOptions may be supplied")
	}
	var opts RequestOptions
	if len(options) == 1 {
		opts = options[0]
	}
	if opts.Timeout < 0 || opts.MaxRetries < 0 {
		return fmt.Errorf("mila: Timeout and MaxRetries must not be negative")
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
		if query {
			timeout = DefaultQueryTimeout
		}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mila: encode %s: %w", endpoint, err)
	}
	path := "/mila/v1/" + endpoint
	refreshed := false
	refreshNext := false
	retries := 0
	for {
		if err := ctx.Err(); err != nil {
			return &NetworkError{Path: path, Err: err}
		}
		authHeader, err := c.auth.authorization(ctx, refreshNext)
		refreshNext = false
		if err != nil {
			return err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("mila: create request: %w", err)
		}
		request.Header = opts.Headers.Clone()
		if request.Header == nil {
			request.Header = make(http.Header)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		if authHeader != "" {
			request.Header.Set("Authorization", authHeader)
		}
		if c.clusterID != "" {
			request.Header.Set("Mila-Cluster-Id", c.clusterID)
		}
		if query && len(opts.Principals) > 0 {
			request.Header.Set("Mila-Principal", strings.Join(opts.Principals, ","))
		}
		response, err := c.httpClient.Do(request)
		if err != nil {
			if ctx.Err() != nil {
				err = ctx.Err()
			}
			if ctx.Err() == nil && retries < opts.MaxRetries {
				retries++
				continue
			}
			return &NetworkError{Path: path, Err: err}
		}
		if c.logger != nil {
			c.logger.DebugContext(ctx, "Mila API response", "path", path, "status", response.StatusCode)
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return decodeResponse(response, path, result)
		}
		responseBody, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			return &NetworkError{Path: path, Err: readErr}
		}
		requestErr := &RequestError{Path: path, StatusCode: response.StatusCode, Body: string(responseBody)}
		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			if !refreshed && (c.auth.iapClientID != "" || c.auth.provider != nil) {
				refreshed = true
				refreshNext = true
				continue
			}
			return &AuthenticationError{Err: requestErr}
		}
		return requestErr
	}
}

func decodeResponse(response *http.Response, path string, result any) error {
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	// BIGINT を含む値を float64 に丸めず保持する。
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return fmt.Errorf("mila: decode %s: %w", path, err)
	}
	if result == nil {
		return fmt.Errorf("mila: decode %s: expected a JSON object, got null", path)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values in response")
		}
		return fmt.Errorf("mila: decode %s: %w", path, err)
	}
	return nil
}
