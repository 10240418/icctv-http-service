package middlewares

import (
	"net/http"
	"strings"
)

// CORSMiddleware 处理 CORS 跨域请求
type CORSMiddleware struct {
	allowedOrigins []string
	allowedMethods []string
	allowedHeaders []string
}

// NewCORSMiddleware 创建 CORS 中间件
// 默认允许所有来源（开发环境），生产环境建议指定具体域名
func NewCORSMiddleware() *CORSMiddleware {
	return &CORSMiddleware{
		allowedOrigins: []string{"*"}, // 允许所有来源，生产环境应指定具体域名
		allowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		allowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
	}
}

// Handle 处理 CORS 的中间件函数
func (c *CORSMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// 处理预检请求（OPTIONS）
		if r.Method == "OPTIONS" {
			c.handlePreflight(w, r, origin)
			return
		}

		// 处理实际请求
		c.setCORSHeaders(w, origin)
		next.ServeHTTP(w, r)
	})
}

// handlePreflight 处理预检请求
func (c *CORSMiddleware) handlePreflight(w http.ResponseWriter, r *http.Request, origin string) {
	c.setCORSHeaders(w, origin)
	w.WriteHeader(http.StatusNoContent)
}

// setCORSHeaders 设置 CORS 响应头
func (c *CORSMiddleware) setCORSHeaders(w http.ResponseWriter, origin string) {
	// 设置允许的来源
	if len(c.allowedOrigins) > 0 && c.allowedOrigins[0] == "*" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else if origin != "" && c.isOriginAllowed(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// 设置允许的方法
	if len(c.allowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.allowedMethods, ", "))
	}

	// 设置允许的请求头
	if len(c.allowedHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.allowedHeaders, ", "))
	}

	// 设置预检请求的缓存时间（秒）
	w.Header().Set("Access-Control-Max-Age", "3600")
}

// isOriginAllowed 检查来源是否被允许
func (c *CORSMiddleware) isOriginAllowed(origin string) bool {
	if len(c.allowedOrigins) == 0 || c.allowedOrigins[0] == "*" {
		return true
	}
	for _, allowed := range c.allowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}
