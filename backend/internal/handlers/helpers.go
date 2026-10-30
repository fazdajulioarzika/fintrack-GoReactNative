package handlers

import (
	"net/http"

	"fintrack-backend/internal/httpx"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// pathID membaca :id dari URL dan memvalidasinya sebagai ObjectID.
func pathID(c *gin.Context) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "ID tidak valid")
		return id, false
	}
	return id, true
}
