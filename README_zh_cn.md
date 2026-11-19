<!--
 * @Description: insert description
 * @Author: yangrongxin
 * @Date: 2026-11-18 10:18:08
 * @LastEditors: yangrongxin
 * @LastEditTime: 2026-11-18 11:22:40
-->
# yrx-http

适用于 Go 服务的可复用 HTTP 辅助函数。该包不依赖 Cloudflare 或 JWT；域名认证和令牌验证仍由调用方应用负责。

## 向客户端提供服务

- `WithRequestLogging` 提供服务端的统一处理函数，添加或传递 `X-Request-ID`，并记录方法、路径、状态码、耗时和请求 ID。`RequestID` 从包装后的请求中读取该 ID。

### 调用示例

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

## 实现服务与服务之间的调用

- `NewClient` 提供服务与服务之间的访问能力，创建一个已配置的 JSON HTTP 客户端。`Get`、`Post` 和 `DoJSON` 返回 `ResponseError`，其中保留响应详情或底层错误。

### 调用示例

```go
import (
	yrxhttp "github.cm/rongxin-tech/yrx-http"
)

client := yrxhttp.NewClient(yrxhttp.ClientOptions{
    BaseURL: "https://jwt.example.com",
    Timeout: 5 * time.Second,
})

var result ValidateResponse
err := client.PostJSON(
    ctx,
    "/api/auth/validate",
    nil,
    &result,
    yrxhttp.WithBearerToken(token),
)
```

## 公共 API

- `WriteJSON`、`WriteError` 和 `WriteHTTPError` 可写入格式一致的 JSON 响应。
- `RequireMethod` 检查 HTTP 方法，若不匹配则写入 `405` 响应。
- `ParseBearerToken` 仅解析 `Authorization` 请求头；令牌验证由应用负责。

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

# 路线图

[x] 实现简单的HTTP方法的封装
[] 在使用的过程中抽离更多的方法

# 贡献方式

fork 源码直接改源码，提交Merge Request合并至主分支。