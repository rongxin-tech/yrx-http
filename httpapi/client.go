package httpapi

import (
	"bytes"         // 暂存编码后的 JSON，并检查响应体是否为空。
	"context"       // 将调用方取消和截止时间传递给 HTTP 请求。
	"encoding/json" // 编码请求体并解码 JSON 响应。
	"fmt"           // 构造包含底层错误的可读错误文本。
	"io"            // 创建请求体读取器并读取响应体。
	"net/http"      // 创建和执行 HTTP 请求，并处理响应。
	"net/url"       // 解析基础 URL 和相对请求路径。
	"strings"       // 拼接 URL 并格式化响应错误。
	"time"          // 配置 HTTP 客户端超时时间。
)

// ClientOptions 配置 Client 的传输、地址、超时和默认请求头。
type ClientOptions struct {
	HTTPClient  *http.Client  // 可选的底层 HTTP 客户端；为空时创建默认客户端。
	BaseURL     string        // 可选基础 URL；请求路径会附加在其路径后。
	Timeout     time.Duration // 请求超时；为零时不额外设置超时。
	Headers     http.Header   // 每个请求默认附带的请求头。
	BearerToken string        // 非空时自动设置 Authorization: Bearer <token>。
	RequestID   string        // 可选默认请求 ID；单次请求传入的 ID 优先。
}

// Client 使用统一配置发送 JSON HTTP 请求。
type Client struct {
	httpClient  *http.Client
	baseURL     string
	headers     http.Header
	bearerToken string
	requestID   string
}

// NewClient 根据 options 创建客户端。传入 HTTPClient 时会保留其传输配置；
// 若同时指定 Timeout，则复制客户端并仅覆盖副本的超时值。
/**
 * *Client 返回一个指向Client结构体的指针
 * 拿到的是对象的引用地址，而不是结构体的一份完整副本
 */
func NewClient(options ClientOptions) *Client {
	hc := options.HTTPClient
	if hc == nil {
		// 获取这个创建的结构体的指针
		hc = &http.Client{Timeout: options.Timeout}
	} else if options.Timeout > 0 {
		// 把指针指向的结构体内容复制一份
		copyClient := *hc
		copyClient.Timeout = options.Timeout
		hc = &copyClient
	}
	return &Client{httpClient: hc, baseURL: strings.TrimRight(options.BaseURL, "/"), headers: options.Headers.Clone(), bearerToken: options.BearerToken, requestID: options.RequestID}
}

// ResponseError 描述请求失败，并保留 HTTP 状态、响应体或底层错误。
type ResponseError struct {
	StatusCode int    // 收到 HTTP 响应时的状态码；网络错误时为零。
	Status     string // HTTP 状态文本。
	Body       []byte // 服务端返回的原始响应体。
	Err        error  // 网络、响应读取或 JSON 解码错误。
}

// Error 返回包含状态或底层错误信息的描述文本。
func (e *ResponseError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("http request failed: %s: %s", e.Status, strings.TrimSpace(string(e.Body)))
	}
	return fmt.Sprintf("http request failed: %v", e.Err)
}

// Unwrap 暴露底层错误，供 errors.Is 和 errors.As 检查。
func (e *ResponseError) Unwrap() error { return e.Err }

// Get 发送 GET 请求，并将非空 JSON 响应解码到 output。
func (c *Client) Get(ctx context.Context, path string, output any) error {
	return c.DoJSON(ctx, http.MethodGet, path, nil, output, "")
}

// Post 将 input 编码为 JSON 发送，并将非空 JSON 响应解码到 output。
func (c *Client) Post(ctx context.Context, path string, input, output any) error {
	return c.DoJSON(ctx, http.MethodPost, path, input, output, "")
}

// DoJSON 使用 method 和 path 发送 JSON 请求。input 或 output 可以为 nil；
// requestID 为空时使用 ClientOptions 中配置的默认请求 ID。
func (c *Client) DoJSON(ctx context.Context, method, path string, input, output any, requestID string) error {
	if requestID == "" {
		requestID = c.requestID
	}
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return &ResponseError{Err: err}
		}
		body = bytes.NewReader(encoded)
	}
	target := path
	if c.baseURL != "" {
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			return &ResponseError{Err: fmt.Errorf("absolute URL not allowed with base URL")}
		}
		base, err := url.Parse(c.baseURL)
		if err != nil {
			return &ResponseError{Err: err}
		}
		requestURL, err := url.Parse(path)
		if err != nil {
			return &ResponseError{Err: err}
		}
		base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(requestURL.Path, "/")
		base.RawQuery = requestURL.RawQuery
		target = base.String()
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return &ResponseError{Err: err}
	}
	for key, values := range c.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}
	if requestID != "" {
		req.Header.Set(RequestIDHeader, requestID)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &ResponseError{Err: err}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ResponseError{StatusCode: resp.StatusCode, Status: resp.Status, Err: err}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &ResponseError{StatusCode: resp.StatusCode, Status: resp.Status, Body: data}
	}
	if output == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, output); err != nil {
		return &ResponseError{StatusCode: resp.StatusCode, Status: resp.Status, Body: data, Err: err}
	}
	return nil
}
