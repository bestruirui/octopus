package middleware

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

func StaticEmbed(urlPrefix string, embedFS fs.FS) gin.HandlerFunc {
	fs := http.FS(embedFS)
	return static(urlPrefix, fs)
}

func StaticLocal(urlPrefix string, localPath string) gin.HandlerFunc {
	fs := http.Dir(localPath)
	return static(urlPrefix, fs)
}

func static(urlPrefix string, fileSystem http.FileSystem) gin.HandlerFunc {
	fileserver := http.FileServer(fileSystem)
	if urlPrefix != "" {
		fileserver = http.StripPrefix(urlPrefix, fileserver)
	}
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Next()
			return
		}
		if strings.HasPrefix(c.Request.URL.Path, "/api") || strings.HasPrefix(c.Request.URL.Path, "/v1") {
			c.Next()
			return
		}

		requestPath := c.Request.URL.Path
		if requestPath == "" {
			requestPath = "/"
		}
		if _, err := fileSystem.Open(requestPath); err == nil {
			switch {
			case requestPath == "/sw.js":
				// Service worker scripts must revalidate on every navigation so clients can pick up updates.
				c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			case requestPath == "/" || path.Ext(requestPath) == ".html":
				// HTML entrypoints should never be immutable, otherwise users can get stuck on an old app shell.
				c.Header("Cache-Control", "no-cache")
			default:
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileserver.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

		acceptsHTML := strings.Contains(c.GetHeader("Accept"), "text/html")
		hasExtension := path.Ext(requestPath) != ""
		if !acceptsHTML && hasExtension {
			c.Next()
			return
		}

		indexFile, err := fileSystem.Open("/index.html")
		if err != nil {
			indexFile, err = fileSystem.Open("index.html")
			if err != nil {
				c.Next()
				return
			}
		}
		_ = indexFile.Close()

		c.Request.URL.Path = "/index.html"
		c.Header("Cache-Control", "no-cache")
		fileserver.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}
