<!--
 * @Description: insert description
 * @Author: yangrongxin
 * @Date: 2026-11-18 10:18:08
 * @LastEditors: yangrongxin
 * @LastEditTime: 2026-11-18 11:22:40
-->
# yrx-http

// Package httpapi 提供与业务领域无关的 HTTP 服务端和客户端能力。
//
// 服务端包含统一 JSON 响应、结构化错误、方法检查、Bearer 头解析、请求
// ID 生成与透传，以及基于 log/slog 的请求日志中间件。客户端包含基础 URL、
// 超时、默认请求头、Bearer Token、请求 ID 和 JSON 编解码支持。
//
// 本包不依赖 Cloudflare SDK 或 JWT 实现；具体的认证、授权和令牌校验由调用
// 方业务模块负责

Reusable HTTP helpers for Go services. The package has no Cloudflare or JWT dependency; domain authentication and token validation remain with the consuming application.
适用于 Go 服务的可复用 HTTP 辅助函数。该包不依赖 Cloudflare 或 JWT；域名认证和令牌验证仍由调用方应用负责。

## Public API
## 公共 API

- `WriteJSON`, `WriteError`, and `WriteHTTPError` write consistent JSON responses.
- `WriteJSON`、`WriteError` 和 `WriteHTTPError` 可写入格式一致的 JSON 响应。
- `RequireMethod` checks an HTTP method and writes a `405` response on mismatch.
- `RequireMethod` 检查 HTTP 方法，若不匹配则写入 `405` 响应。
- `WithRequestLogging` adds or propagates `X-Request-ID` and logs method, path, status, duration, and request ID. `RequestID` reads the ID from a wrapped request.
- `WithRequestLogging` 添加或传递 `X-Request-ID`，并记录方法、路径、状态码、耗时和请求 ID。`RequestID` 从包装后的请求中读取该 ID。
- `ParseBearerToken` parses an Authorization header only; token validation belongs to the application.
- `ParseBearerToken` 仅解析 `Authorization` 请求头；令牌验证由应用负责。
- `NewClient` creates a configured JSON HTTP client. `Get`, `Post`, and `DoJSON` return `ResponseError`, which retains response details or the underlying error.
- `NewClient` 创建一个已配置的 JSON HTTP 客户端。`Get`、`Post` 和 `DoJSON` 返回 `ResponseError`，其中保留响应详情或底层错误。

```go
import (
    "net/http"

    httpapi "github.com/rongxin-tech/yrx-http/httpapi"
)

func handler(w http.ResponseWriter, r *http.Request) {
    if !httpapi.RequireMethod(w, r, http.MethodGet) { return }
    httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
```

## 完整 HTTP 服务示例

下面的程序会启动一个监听 `:8080` 的 HTTP 服务，并提供 `GET /healthz` 接口：

```go
package main

import (
    "log"
    "log/slog"
    "net/http"

    httpapi "github.com/rongxin-tech/yrx-http/httpapi"
)

func health(w http.ResponseWriter, r *http.Request) {
    if !httpapi.RequireMethod(w, r, http.MethodGet) {
        return
    }
    httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", health)

    // 中间件会生成或透传 X-Request-ID，并记录请求日志。
    handler := httpapi.WithRequestLogging(mux, slog.Default())
    addr := ":8080"

    log.Printf("HTTP server listening on http://localhost%s", addr)
    if err := http.ListenAndServe(addr, handler); err != nil {
        log.Fatal(err)
    }
}
```

在一个新的 Go 项目中运行：

```bash
mkdir yrx-http-demo
cd yrx-http-demo
go mod init example.com/yrx-http-demo
go get github.com/rongxin-tech/yrx-http
# 将上面的代码保存为 main.go
go run .
```

另开终端验证服务：

```bash
curl -i http://localhost:8080/healthz
```

响应示例：

```http
HTTP/1.1 200 OK
Content-Type: application/json
X-Request-Id: 7b2e...

{"status":"ok"}
```

Wrap a handler with `httpapi.WithRequestLogging(handler, logger)` to propagate or generate `X-Request-ID` and log method, path, status, duration, and request ID. Use `ParseBearerToken` for standard Authorization header parsing; it does not validate token contents.
使用 `httpapi.WithRequestLogging(handler, logger)` 包装处理程序，以传递或生成 `X-Request-ID`，并记录方法、路径、状态码、耗时和请求 ID。使用 `ParseBearerToken` 解析标准的 Authorization 请求头；它不会验证令牌内容。

For outbound requests, create `httpapi.NewClient(httpapi.ClientOptions{BaseURL: ..., Timeout: ..., Headers: ..., BearerToken: ...})` and call `Get`, `Post`, or `DoJSON`. `ResponseError` retains HTTP status/body or the underlying transport/JSON error.
对于出站请求，创建 `httpapi.NewClient(httpapi.ClientOptions{BaseURL: ..., Timeout: ..., Headers: ..., BearerToken: ...})`，然后调用 `Get`、`Post` 或 `DoJSON`。`ResponseError` 会保留 HTTP 状态码/响应体，或底层传输/JSON 错误。
