package middleware

import (
	"net/url"
	"strings"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowCredentials = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"*"}
	config.ExposeHeaders = []string{"Content-Disposition"}
	// CORS 白名单:
	// - 为空: 仅允许本机来源(localhost/127.0.0.1/::1)，方便前端本地调试
	// - "*": 允许所有来源
	// - 逗号分隔的域名列表: 只允许指定的域名 (如 "https://example.com,https://example2.com")
	config.AllowOriginFunc = func(origin string) bool {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			return false
		}

		allowed, err := op.SettingGetString(model.SettingKeyCORSAllowOrigins)
		if err != nil {
			return isLocalOrigin(origin)
		}
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			return isLocalOrigin(origin)
		}
		if allowed == "*" {
			return true
		}

		for _, item := range strings.Split(allowed, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			// 支持:
			// - 完整 origin: https://example.com
			// - host:port: example.com:3000
			// - 仅域名: example.com
			if isOriginMatched(origin, item) {
				return true
			}
		}
		return false
	}
	return cors.New(config)
}

func isOriginMatched(origin, allowed string) bool {
	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return false
	}

	originScheme := strings.ToLower(parsedOrigin.Scheme)
	originHost := strings.ToLower(parsedOrigin.Host)
	originHostname := strings.ToLower(parsedOrigin.Hostname())
	if originScheme == "" || originHost == "" || originHostname == "" {
		return false
	}

	allowed = strings.ToLower(strings.TrimSpace(strings.TrimRight(allowed, "/")))
	if allowed == "" {
		return false
	}

	// 完整 origin: scheme://host[:port]
	if strings.Contains(allowed, "://") {
		parsedAllowed, err := url.Parse(allowed)
		if err != nil {
			return false
		}
		return originScheme == strings.ToLower(parsedAllowed.Scheme) && originHost == strings.ToLower(parsedAllowed.Host)
	}

	// host:port 或 host
	return allowed == originHost || allowed == originHostname
}

func isLocalOrigin(origin string) bool {
	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsedOrigin.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
