package handlers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fintrack-backend/internal/httpx"
	"fintrack-backend/internal/middleware"
	"fintrack-backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type TransactionHandler struct {
	txs        *mongo.Collection
	categories *mongo.Collection
	users      *mongo.Collection
}

func NewTransactionHandler(db *mongo.Database) *TransactionHandler {
	return &TransactionHandler{
		txs:        db.Collection("transactions"),
		categories: db.Collection("categories"),
		users:      db.Collection("users"),
	}
}

// ---------- Request & response ----------

// Batas nominal ~1 triliun rupiah: mencegah overflow dan input tidak masuk akal.
type createTxReq struct {
	Type            string    `json:"type" binding:"required,oneof=income expense"`
	Amount          int64     `json:"amount" binding:"required,gt=0,lte=999999999999"`
	CategoryID      string    `json:"category_id" binding:"required"`
	TransactionDate time.Time `json:"transaction_date" binding:"required"` // format RFC3339, mis. 2026-10-29T08:30:00+07:00
	PaymentMethod   string    `json:"payment_method" binding:"max=50"`
	Note            string    `json:"note" binding:"max=500"`
}

// Semua field pointer: nil berarti "tidak diubah".
type updateTxReq struct {
	Type            *string    `json:"type" binding:"omitempty,oneof=income expense"`
	Amount          *int64     `json:"amount" binding:"omitempty,gt=0,lte=999999999999"`
	CategoryID      *string    `json:"category_id"`
	TransactionDate *time.Time `json:"transaction_date"`
	PaymentMethod   *string    `json:"payment_method" binding:"omitempty,max=50"`
	Note            *string    `json:"note" binding:"omitempty,max=500"`
}

type categoryDTO struct {
	ID   bson.ObjectID `json:"id"`
	Name string        `json:"name"`
	Icon string        `json:"icon"`
}

type txDTO struct {
	ID              bson.ObjectID `json:"id"`
	Type            string        `json:"type"`
	Amount          int64         `json:"amount"`
	Category        *categoryDTO  `json:"category"`
	Note            string        `json:"note"`
	PaymentMethod   string        `json:"payment_method"`
	TransactionDate time.Time     `json:"transaction_date"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// ---------- Handlers ----------

func (h *TransactionHandler) Create(c *gin.Context) {
	uid := middleware.UserID(c)
	ctx := c.Request.Context()

	var req createTxReq
	if err := c.ShouldBindJSON(&req); err != nil || req.TransactionDate.IsZero() {
		httpx.Error(c, http.StatusBadRequest, "validation_error",
			"Data tidak valid: type (income/expense), amount (bilangan bulat > 0), category_id, dan transaction_date (RFC3339) wajib diisi")
		return
	}

	// Anti-duplikasi: request ulang dengan key yang sama mengembalikan transaksi yang sudah ada.
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if len(key) > 100 {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "Idempotency-Key maksimal 100 karakter")
		return
	}
	if key != "" {
		if existing, found := h.findByKey(ctx, uid, key); found {
			h.respondOne(c, http.StatusOK, existing)
			return
		}
	}

	catID, err := bson.ObjectIDFromHex(req.CategoryID)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "category_id tidak valid")
		return
	}
	if status, msg := h.checkCategory(ctx, uid, catID, req.Type); status != 0 {
		httpx.Error(c, status, "validation_error", msg)
		return
	}

	now := time.Now().UTC()
	tx := models.Transaction{
		ID:              bson.NewObjectID(),
		UserID:          uid,
		Type:            req.Type,
		Amount:          req.Amount,
		CategoryID:      catID,
		Note:            strings.TrimSpace(req.Note),
		PaymentMethod:   strings.TrimSpace(req.PaymentMethod),
		TransactionDate: req.TransactionDate.UTC(),
		IdempotencyKey:  key,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if _, err := h.txs.InsertOne(ctx, tx); err != nil {
		// Dua request dengan key sama yang masuk bersamaan: yang kedua kena unique index.
		if mongo.IsDuplicateKeyError(err) && key != "" {
			if existing, found := h.findByKey(ctx, uid, key); found {
				h.respondOne(c, http.StatusOK, existing)
				return
			}
		}
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	h.respondOne(c, http.StatusCreated, &tx)
}

func (h *TransactionHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var tx models.Transaction
	err := h.txs.FindOne(c.Request.Context(), bson.M{"_id": id, "user_id": middleware.UserID(c)}).Decode(&tx)
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.Error(c, http.StatusNotFound, "not_found", "Transaksi tidak ditemukan")
		return
	}
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	h.respondOne(c, http.StatusOK, &tx)
}

func (h *TransactionHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	uid := middleware.UserID(c)
	ctx := c.Request.Context()

	var req updateTxReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "Data tidak valid")
		return
	}

	var tx models.Transaction
	err := h.txs.FindOne(ctx, bson.M{"_id": id, "user_id": uid}).Decode(&tx)
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.Error(c, http.StatusNotFound, "not_found", "Transaksi tidak ditemukan")
		return
	}
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	if req.Type != nil {
		tx.Type = *req.Type
	}
	if req.Amount != nil {
		tx.Amount = *req.Amount
	}
	if req.CategoryID != nil {
		catID, err := bson.ObjectIDFromHex(*req.CategoryID)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "validation_error", "category_id tidak valid")
			return
		}
		tx.CategoryID = catID
	}
	if req.TransactionDate != nil {
		if req.TransactionDate.IsZero() {
			httpx.Error(c, http.StatusBadRequest, "validation_error", "transaction_date tidak valid")
			return
		}
		tx.TransactionDate = req.TransactionDate.UTC()
	}
	if req.PaymentMethod != nil {
		tx.PaymentMethod = strings.TrimSpace(*req.PaymentMethod)
	}
	if req.Note != nil {
		tx.Note = strings.TrimSpace(*req.Note)
	}

	// Jika jenis atau kategori berubah, pastikan keduanya masih cocok.
	if req.Type != nil || req.CategoryID != nil {
		if status, msg := h.checkCategory(ctx, uid, tx.CategoryID, tx.Type); status != 0 {
			httpx.Error(c, status, "validation_error", msg)
			return
		}
	}

	tx.UpdatedAt = time.Now().UTC()
	_, err = h.txs.UpdateOne(ctx, bson.M{"_id": id, "user_id": uid}, bson.M{"$set": bson.M{
		"type":             tx.Type,
		"amount":           tx.Amount,
		"category_id":      tx.CategoryID,
		"note":             tx.Note,
		"payment_method":   tx.PaymentMethod,
		"transaction_date": tx.TransactionDate,
		"updated_at":       tx.UpdatedAt,
	}})
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	h.respondOne(c, http.StatusOK, &tx)
}

func (h *TransactionHandler) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	res, err := h.txs.DeleteOne(c.Request.Context(), bson.M{"_id": id, "user_id": middleware.UserID(c)})
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	if res.DeletedCount == 0 {
		httpx.Error(c, http.StatusNotFound, "not_found", "Transaksi tidak ditemukan")
		return
	}
	c.Status(http.StatusNoContent)
}

// List: GET /transactions?type=&category_id=&start_date=&end_date=&q=&page=&limit=&sort=
func (h *TransactionHandler) List(c *gin.Context) {
	uid := middleware.UserID(c)
	ctx := c.Request.Context()
	filter := bson.M{"user_id": uid} // user_id selalu dari token

	if t := c.Query("type"); t != "" {
		if t != models.TypeIncome && t != models.TypeExpense {
			httpx.Error(c, http.StatusBadRequest, "validation_error", "type harus income atau expense")
			return
		}
		filter["type"] = t
	}
	if s := c.Query("category_id"); s != "" {
		oid, err := bson.ObjectIDFromHex(s)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "validation_error", "category_id tidak valid")
			return
		}
		filter["category_id"] = oid
	}

	// Tanggal: "2026-10-01" (dihitung menurut zona waktu user) atau RFC3339 lengkap.
	loc := h.userLocation(ctx, uid)
	dateFilter := bson.M{}
	if s := c.Query("start_date"); s != "" {
		t, _, err := parseDateParam(s, loc)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "validation_error", "start_date tidak valid (YYYY-MM-DD atau RFC3339)")
			return
		}
		dateFilter["$gte"] = t.UTC()
	}
	if s := c.Query("end_date"); s != "" {
		t, dateOnly, err := parseDateParam(s, loc)
		if err != nil {
			httpx.Error(c, http.StatusBadRequest, "validation_error", "end_date tidak valid (YYYY-MM-DD atau RFC3339)")
			return
		}
		if dateOnly {
			dateFilter["$lt"] = t.AddDate(0, 0, 1).UTC() // end_date inklusif sampai akhir hari
		} else {
			dateFilter["$lte"] = t.UTC()
		}
	}
	if len(dateFilter) > 0 {
		filter["transaction_date"] = dateFilter
	}

	if q := strings.TrimSpace(c.Query("q")); q != "" {
		if r := []rune(q); len(r) > 50 {
			q = string(r[:50])
		}
		// QuoteMeta: input user diperlakukan sebagai teks biasa, bukan pola regex.
		filter["note"] = bson.M{"$regex": regexp.QuoteMeta(q), "$options": "i"}
	}

	page := intQuery(c, "page", 1, 1, 1_000_000)
	limit := intQuery(c, "limit", 20, 1, 100)

	sort := bson.D{{Key: "transaction_date", Value: -1}, {Key: "_id", Value: -1}}
	switch c.Query("sort") {
	case "date_asc":
		sort = bson.D{{Key: "transaction_date", Value: 1}, {Key: "_id", Value: 1}}
	case "amount_desc":
		sort = bson.D{{Key: "amount", Value: -1}, {Key: "_id", Value: -1}}
	case "amount_asc":
		sort = bson.D{{Key: "amount", Value: 1}, {Key: "_id", Value: 1}}
	}

	total, err := h.txs.CountDocuments(ctx, filter)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	cur, err := h.txs.Find(ctx, filter, options.Find().
		SetSort(sort).
		SetSkip(int64((page-1)*limit)).
		SetLimit(int64(limit)))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	var txs []models.Transaction
	if err := cur.All(ctx, &txs); err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	dtos, err := h.toDTOs(ctx, txs)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": dtos,
		"meta": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": int(math.Ceil(float64(total) / float64(limit))),
		},
	})
}

// ---------- Helper ----------

// checkCategory memastikan kategori ada, boleh dipakai user ini, dan jenisnya cocok.
// Mengembalikan status 0 jika valid.
func (h *TransactionHandler) checkCategory(ctx context.Context, uid, catID bson.ObjectID, txType string) (int, string) {
	var cat models.Category
	err := h.categories.FindOne(ctx, bson.M{
		"_id": catID,
		"$or": bson.A{bson.M{"user_id": nil}, bson.M{"user_id": uid}},
	}).Decode(&cat)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return http.StatusBadRequest, "Kategori tidak ditemukan"
	}
	if err != nil {
		return http.StatusInternalServerError, "Terjadi kesalahan server"
	}
	if cat.Type != txType {
		return http.StatusBadRequest, "Kategori tidak sesuai dengan jenis transaksi"
	}
	return 0, ""
}

func (h *TransactionHandler) findByKey(ctx context.Context, uid bson.ObjectID, key string) (*models.Transaction, bool) {
	var tx models.Transaction
	err := h.txs.FindOne(ctx, bson.M{"user_id": uid, "idempotency_key": key}).Decode(&tx)
	if err != nil {
		return nil, false
	}
	return &tx, true
}

func (h *TransactionHandler) respondOne(c *gin.Context, status int, tx *models.Transaction) {
	dtos, err := h.toDTOs(c.Request.Context(), []models.Transaction{*tx})
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	c.JSON(status, gin.H{"transaction": dtos[0]})
}

// toDTOs menyertakan nama + ikon kategori dengan satu query (bukan satu query per transaksi).
func (h *TransactionHandler) toDTOs(ctx context.Context, txs []models.Transaction) ([]txDTO, error) {
	out := make([]txDTO, 0, len(txs))
	if len(txs) == 0 {
		return out, nil
	}

	seen := map[bson.ObjectID]bool{}
	ids := bson.A{}
	for _, t := range txs {
		if !seen[t.CategoryID] {
			seen[t.CategoryID] = true
			ids = append(ids, t.CategoryID)
		}
	}
	cur, err := h.categories.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	var cats []models.Category
	if err := cur.All(ctx, &cats); err != nil {
		return nil, err
	}
	byID := make(map[bson.ObjectID]models.Category, len(cats))
	for _, cat := range cats {
		byID[cat.ID] = cat
	}

	for _, t := range txs {
		dto := txDTO{
			ID:              t.ID,
			Type:            t.Type,
			Amount:          t.Amount,
			Note:            t.Note,
			PaymentMethod:   t.PaymentMethod,
			TransactionDate: t.TransactionDate,
			CreatedAt:       t.CreatedAt,
			UpdatedAt:       t.UpdatedAt,
		}
		if cat, ok := byID[t.CategoryID]; ok {
			dto.Category = &categoryDTO{ID: cat.ID, Name: cat.Name, Icon: cat.Icon}
		}
		out = append(out, dto)
	}
	return out, nil
}

func (h *TransactionHandler) userLocation(ctx context.Context, uid bson.ObjectID) *time.Location {
	var u struct {
		Timezone string `bson:"timezone"`
	}
	err := h.users.FindOne(ctx, bson.M{"_id": uid},
		options.FindOne().SetProjection(bson.M{"timezone": 1})).Decode(&u)
	if err == nil {
		if loc, err := time.LoadLocation(u.Timezone); err == nil {
			return loc
		}
	}
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.UTC
}

func parseDateParam(s string, loc *time.Location) (t time.Time, dateOnly bool, err error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, false, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		return t, true, nil
	}
	return time.Time{}, false, fmt.Errorf("format tanggal tidak dikenali: %q", s)
}

func intQuery(c *gin.Context, name string, def, min, max int) int {
	v, err := strconv.Atoi(c.Query(name))
	if err != nil {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}