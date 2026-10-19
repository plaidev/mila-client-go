package mila

import "fmt"

// RequestError は API が返した成功以外の HTTP レスポンスを保持する。
type RequestError struct {
	Path       string
	StatusCode int
	Body       string
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("mila: %s returned HTTP %d: %s", e.Path, e.StatusCode, e.Body)
}

// AuthenticationError はトークン取得失敗、または HTTP 401/403 を表す。
// HTTP エラーの場合は errors.As で RequestError も取得できる。
type AuthenticationError struct{ Err error }

func (e *AuthenticationError) Error() string { return "mila: authentication failed: " + e.Err.Error() }
func (e *AuthenticationError) Unwrap() error { return e.Err }

// NetworkError は通信失敗を表す。errors.Is で context の中断・期限超過を判定できる。
type NetworkError struct {
	Path string
	Err  error
}

func (e *NetworkError) Error() string { return fmt.Sprintf("mila: %s: %v", e.Path, e.Err) }
func (e *NetworkError) Unwrap() error { return e.Err }
