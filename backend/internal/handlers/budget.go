package handlers

import (
	"context"
	"errors"
	"math"
	"net/http"
	"sort"
	"time"

	"fintrack-backend/internal/httpx"
	"fintrack-backend/internal/middleware"
	"fintrack-backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type BudgetHandler struct {
	budgets    *mongo.Collection
	txs        *mongo.Collection
	categories *mongo.Collection
	users      *mongo.Collection
}

func NewBudgetHandler(db *mongo.Database) *BudgetHandler {
	return &BudgetHandler{
		budgets:    db.Collection("budgets"),
		txs:        db.Collection("transactions"),
		categories: db.Collection("categories"),
		users:      db.Collection("users"),
	}
}

type createBudgetReq struct {
	CategoryID  string `json:"category_id" binding:"required"`
	Period      string `json:"period" binding:"required"` // "YYYY-MM"
	LimitAmount int64  `json:"limit_amount" binding:"required,gt=0,lte=999999999999"`
}

type updateBudgetReq struct {
	LimitAmount int64 `json:"limit_amount" binding:"required,gt=0,lte=999999999999"`
}

type budgetDTO struct {
	ID              bson.ObjectID `json:"id"`
	Category        *categoryDTO  `json:"category"`
	Period          string        `json:"period"`
	LimitAmount     int64         `json:"limit_amount"`
	SpentAmount     int64         `json:"spent_amount"`
	RemainingAmount int64         `json:"remaining_amount"` // negatif jika terlampaui
	UsagePercent    float64       `json:"usage_percent"`
	Status          string        `json:"status"` // normal | warning | exceeded
}

// List: GET /budgets?month=2026-10
func (h *BudgetHandler) List(c *gin.Context) {
	uid := middleware.UserID(c)
	ctx := c.Request.Context()
	loc := loadUserLocation(ctx, h.users, uid)

	p, err := parseMonth(c.Query("month"), loc)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "month harus berformat YYYY-MM, contoh 2026-10")
		return
	}

	cur, err := h.budgets.Find(ctx, bson.M{"user_id": uid, "period": p.Label})
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	var list []models.Budget
	if err := cur.All(ctx, &list); err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	dtos, err := h.buildDTOs(ctx, uid, loc, list)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	sort.Slice(dtos, func(i, j int) bool { return dtos[i].UsagePercent > dtos[j].UsagePercent })

	var totalLimit, totalSpent int64
	for _, d := range dtos {
		totalLimit += d.LimitAmount
		totalSpent += d.SpentAmount
	}
	c.JSON(http.StatusOK, gin.H{
		"period":      periodJSON(p, loc),
		"total_limit": totalLimit,
		"total_spent": totalSpent,
		"data":        dtos,
	})
}

func (h *BudgetHandler) Create(c *gin.Context) {
	uid := middleware.UserID(c)
	ctx := c.Request.Context()

	var req createBudgetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error",
			"Data tidak valid: category_id, period (YYYY-MM), dan limit_amount (bilangan bulat > 0) wajib diisi")
		return
	}
	loc := loadUserLocation(ctx, h.users, uid)
	p, err := parseMonth(req.Period, loc)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "period harus berformat YYYY-MM, contoh 2026-10")
		return
	}
	catID, err := bson.ObjectIDFromHex(req.CategoryID)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "category_id tidak valid")
		return
	}

	// Kategori harus ada, boleh dipakai user ini, dan bertipe expense
	var cat models.Category
	err = h.categories.FindOne(ctx, bson.M{
		"_id": catID,
		"$or": bson.A{bson.M{"user_id": nil}, bson.M{"user_id": uid}},
	}).Decode(&cat)
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "Kategori tidak ditemukan")
		return
	}
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	if cat.Type != models.TypeExpense {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "Anggaran hanya untuk kategori pengeluaran")
		return
	}

	now := time.Now().UTC()
	b := models.Budget{
		ID:          bson.NewObjectID(),
		UserID:      uid,
		CategoryID:  catID,
		Period:      p.Label,
		LimitAmount: req.LimitAmount,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := h.budgets.InsertOne(ctx, b); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			httpx.Error(c, http.StatusConflict, "budget_exists", "Anggaran untuk kategori dan bulan ini sudah ada")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	h.respondOne(c, http.StatusCreated, loc, b)
}

func (h *BudgetHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	uid := middleware.UserID(c)
	ctx := c.Request.Context()

	var req updateBudgetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "limit_amount harus bilangan bulat lebih dari 0")
		return
	}

	var b models.Budget
	err := h.budgets.FindOneAndUpdate(ctx,
		bson.M{"_id": id, "user_id": uid},
		bson.M{"$set": bson.M{"limit_amount": req.LimitAmount, "updated_at": time.Now().UTC()}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&b)
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.Error(c, http.StatusNotFound, "not_found", "Anggaran tidak ditemukan")
		return
	}
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	h.respondOne(c, http.StatusOK, loadUserLocation(ctx, h.users, uid), b)
}

func (h *BudgetHandler) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	res, err := h.budgets.DeleteOne(c.Request.Context(), bson.M{"_id": id, "user_id": middleware.UserID(c)})
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	if res.DeletedCount == 0 {
		httpx.Error(c, http.StatusNotFound, "not_found", "Anggaran tidak ditemukan")
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------- Helper ----------

func (h *BudgetHandler) respondOne(c *gin.Context, status int, loc *time.Location, b models.Budget) {
	dtos, err := h.buildDTOs(c.Request.Context(), b.UserID, loc, []models.Budget{b})
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	c.JSON(status, gin.H{"budget": dtos[0]})
}

// buildDTOs menghitung pemakaian dari transaksi expense pada bulan anggaran.
// Semua budget dalam satu panggilan harus memiliki period yang sama.
func (h *BudgetHandler) buildDTOs(ctx context.Context, uid bson.ObjectID, loc *time.Location, budgets []models.Budget) ([]budgetDTO, error) {
	out := make([]budgetDTO, 0, len(budgets))
	if len(budgets) == 0 {
		return out, nil
	}

	p, err := parseMonth(budgets[0].Period, loc)
	if err != nil {
		return nil, err
	}
	ids := bson.A{}
	for _, b := range budgets {
		ids = append(ids, b.CategoryID)
	}

	// Hanya pengeluaran yang dihitung; pemasukan tidak mengurangi anggaran
	cur, err := h.txs.Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{
			"user_id":          uid,
			"type":             models.TypeExpense,
			"category_id":      bson.M{"$in": ids},
			"transaction_date": bson.M{"$gte": p.Start, "$lt": p.End},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   "$category_id",
			"total": bson.M{"$sum": "$amount"},
		}}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		CategoryID bson.ObjectID `bson:"_id"`
		Total      int64         `bson:"total"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	spent := make(map[bson.ObjectID]int64, len(rows))
	for _, r := range rows {
		spent[r.CategoryID] = r.Total
	}

	cc, err := h.categories.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	var cats []models.Category
	if err := cc.All(ctx, &cats); err != nil {
		return nil, err
	}
	catByID := make(map[bson.ObjectID]models.Category, len(cats))
	for _, cat := range cats {
		catByID[cat.ID] = cat
	}

	for _, b := range budgets {
		s := spent[b.CategoryID]
		dto := budgetDTO{
			ID:              b.ID,
			Period:          b.Period,
			LimitAmount:     b.LimitAmount,
			SpentAmount:     s,
			RemainingAmount: b.LimitAmount - s,
			UsagePercent:    math.Round(float64(s)/float64(b.LimitAmount)*1000) / 10,
			Status:          budgetStatus(s, b.LimitAmount),
		}
		if cat, ok := catByID[b.CategoryID]; ok {
			dto.Category = &categoryDTO{ID: cat.ID, Name: cat.Name, Icon: cat.Icon}
		}
		out = append(out, dto)
	}
	return out, nil
}

// budgetStatus memakai perbandingan integer agar tidak terkena kesalahan pembulatan float.
func budgetStatus(spent, limit int64) string {
	switch {
	case spent >= limit:
		return "exceeded"
	case spent*100 >= limit*80:
		return "warning"
	default:
		return "normal"
	}
}
