// Package middleware berisi fungsi yang berjalan sebelum handler.
package middleware

import (
	"net/http"
	"strings"

	"fintrack-backend/internal/auth"
	"fintrack-backend/internal/httpx"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const UserIDKey = "userID"

// RequireAuth menolak request tanpa access token valid, lalu menyimpan user ID ke context.
// user_id SELALU diambil dari token, tidak pernah dari body/query (PRD bagian 12).
func RequireAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			httpx.Error(c, http.StatusUnauthorized, "unauthorized", "Token autentikasi diperlukan")
			return
		}
		userIDHex, err := auth.ParseAccessToken(strings.TrimPrefix(header, "Bearer "), secret)
		if err != nil {
			httpx.Error(c, http.StatusUnauthorized, "unauthorized", "Token tidak valid atau kedaluwarsa")
			return
		}
		oid, err := bson.ObjectIDFromHex(userIDHex)
		if err != nil {
			httpx.Error(c, http.StatusUnauthorized, "unauthorized", "Token tidak valid")
			return
		}
		c.Set(UserIDKey, oid)
		c.Next()
	}
}

// UserID dipakai handler untuk mengambil ID user yang sedang login.
func UserID(c *gin.Context) bson.ObjectID {
	return c.MustGet(UserIDKey).(bson.ObjectID)
}
