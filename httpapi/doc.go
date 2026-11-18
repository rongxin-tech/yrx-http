// Package httpapi 提供与业务领域无关的 HTTP 服务端和客户端能力。
//
// 服务端包含统一 JSON 响应、结构化错误、方法检查、Bearer 头解析、请求
// ID 生成与透传，以及基于 log/slog 的请求日志中间件。客户端包含基础 URL、
// 超时、默认请求头、Bearer Token、请求 ID 和 JSON 编解码支持。
//
// 本包不依赖 Cloudflare SDK 或 JWT 实现；具体的认证、授权和令牌校验由调用
// 方业务模块负责。
package httpapi
