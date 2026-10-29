// Package httpx berisi helper respons agar format error konsisten di seluruh API.
package httpx

import "github.com/gin-gonic/gin"

// Error menulis respons: {"error": {"code": "...", "message": "..."}}
func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{"code": code, "message": message},
	})
}
