package handlers

import (
	"net/http"

	"fintrack-backend/internal/httpx"
	"fintrack-backend/internal/middleware"
	"fintrack-backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type CategoryHandler struct {
	categories *mongo.Collection
}

func NewCategoryHandler(db *mongo.Database) *CategoryHandler {
	return &CategoryHandler{categories: db.Collection("categories")}
}

// List: GET /categories?type=income|expense
// Mengembalikan kategori sistem + kategori milik user yang sedang login.
func (h *CategoryHandler) List(c *gin.Context) {
	filter := bson.M{"$or": bson.A{
		bson.M{"user_id": nil},
		bson.M{"user_id": middleware.UserID(c)},
	}}

	if t := c.Query("type"); t != "" {
		if t != models.TypeIncome && t != models.TypeExpense {
			httpx.Error(c, http.StatusBadRequest, "validation_error", "type harus income atau expense")
			return
		}
		filter["type"] = t
	}

	cur, err := h.categories.Find(c.Request.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "type", Value: 1}, {Key: "sort_order", Value: 1}}))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	cats := []models.Category{}
	if err := cur.All(c.Request.Context(), &cats); err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cats})
}
