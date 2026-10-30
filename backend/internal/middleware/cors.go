package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS mengizinkan browser dari origin tertentu memanggil API.
// Aplikasi native (HP) tidak memakai CORS, jadi ini hanya relevan untuk mode web.
func CORS(allowed []string) gin.HandlerFunc {
	set := make(map[string]bool, len(allowed))
	for _, o := range allowed {
		set[o] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && set[origin] {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Add("Vary", "Origin")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Max-Age", "600")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
