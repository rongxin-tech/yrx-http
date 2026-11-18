package httpapi

import (
	"context"       // 将请求 ID 放入请求上下文并传递给下游处理器。
	"crypto/rand"   // 生成不可预测的请求 ID。
	"encoding/hex"  // 将请求 ID 的随机字节编码为 HTTP 头可用的文本。
	"encoding/json" // 编码 JSON 响应体。
	"errors"        // 判断错误链中是否包含结构化 HTTP 错误。
	"log/slog"      // 输出可注入的结构化请求日志。
	"net/http"      // 提供 HTTP 请求、响应及状态码定义。
	"strings"       // 解析 Bearer 头并规范化请求 ID。
	"time"          // 记录请求耗时并为随机源故障提供备用 ID。
)

// RequestIDHeader 是请求 ID 在 HTTP 请求和响应中使用的头名称。
const RequestIDHeader = "X-Request-ID"

type contextKey struct{}

// Error is an HTTP error with a client-safe message and response status.
type Error struct {
	Status  int    `json:"-"`     // HTTP 响应状态码。
	Message string `json:"error"` // 返回给客户端的错误文本。
}

// Error 返回适合记录或包装的错误文本。
func (e *Error) Error() string { return e.Message }

// NewError 创建指定状态码和客户端可见文本的结构化 HTTP 错误。
func NewError(status int, message string) *Error { return &Error{Status: status, Message: message} }

// WriteJSON 设置 JSON Content-Type，写入状态码，并将 value 编码到响应体。
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// WriteError 以统一的 {"error":"..."} JSON 格式写入错误响应。
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, NewError(status, message))
}

// WriteHTTPError maps structured errors to their configured status. Unknown
// errors are returned as 500 without exposing their details.
func WriteHTTPError(w http.ResponseWriter, err error) {
	var httpErr *Error
	if errors.As(err, &httpErr) {
		WriteError(w, httpErr.Status, httpErr.Message)
		return
	}
	WriteError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
}

// RequireMethod 检查请求方法；不匹配时写入 405 和 Allow 头并返回 false。
func RequireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	WriteError(w, http.StatusMethodNotAllowed, "method must be "+method)
	return false
}

// ParseBearerToken 从 Authorization 头提取 Bearer 凭据，不验证令牌内容。
// 方案名称大小写不敏感，头部必须恰好包含方案和一个令牌字段。
func ParseBearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("Authorization header must be Bearer <token>")
	}
	return parts[1], nil
}

// RequestID 返回中间件放入请求上下文的请求 ID；若未经过中间件，则读取请求头。
func RequestID(r *http.Request) string {
	if id, ok := r.Context().Value(contextKey{}).(string); ok {
		return id
	}
	return r.Header.Get(RequestIDHeader)
}

// WithRequestLogging assigns/propagates a request ID and logs method, path,
// status, duration, and request_id for every request.
func WithRequestLogging(next http.Handler, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get(RequestIDHeader))
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(RequestIDHeader, id)
		started := time.Now()
		recorder := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(recorder, r.WithContext(context.WithValue(r.Context(), contextKey{}, id)))
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		logger.Info("http request", "method", r.Method, "path", r.URL.Path, "status", status, "duration", time.Since(started), "request_id", id)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b[:])
}
