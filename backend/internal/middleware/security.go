package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders: respons API berisi data keuangan, jadi jangan sampai disimpan di cache perantara.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Cache-Control", "no-store")
		h.Set("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

// MaxBody membatasi ukuran body request. Body yang lebih besar akan gagal dibaca (dibalas 400).
func MaxBody(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		c.Next()
	}
}
