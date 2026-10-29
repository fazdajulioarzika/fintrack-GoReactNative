package handlers

import (
	"context"
	"math"
	"net/http"

	"fintrack-backend/internal/httpx"
	"fintrack-backend/internal/middleware"
	"fintrack-backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type DashboardHandler struct {
	txs        *mongo.Collection
	categories *mongo.Collection
	users      *mongo.Collection
}

func NewDashboardHandler(db *mongo.Database) *DashboardHandler {
	return &DashboardHandler{
		txs:        db.Collection("transactions"),
		categories: db.Collection("categories"),
		users:      db.Collection("users"),
	}
}

// totals menjumlahkan pemasukan dan pengeluaran untuk filter tertentu.
func (h *DashboardHandler) totals(ctx context.Context, match bson.M) (income, expense, count int64, err error) {
	cur, err := h.txs.Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   "$type",
			"total": bson.M{"$sum": "$amount"},
			"count": bson.M{"$sum": 1},
		}}},
	})
	if err != nil {
		return
	}
	var rows []struct {
		Type  string `bson:"_id"`
		Total int64  `bson:"total"`
		Count int64  `bson:"count"`
	}
	if err = cur.All(ctx, &rows); err != nil {
		return
	}
	for _, r := range rows {
		count += r.Count
		switch r.Type {
		case models.TypeIncome:
			income = r.Total
		case models.TypeExpense:
			expense = r.Total
		}
	}
	return
}

// Summary: GET /dashboard/summary?month=2026-10
func (h *DashboardHandler) Summary(c *gin.Context) {
	uid := middleware.UserID(c)
	ctx := c.Request.Context()
	loc := loadUserLocation(ctx, h.users, uid)

	p, err := parseMonth(c.Query("month"), loc)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "month harus berformat YYYY-MM, contoh 2026-10")
		return
	}

	// Saldo = semua pemasukan - semua pengeluaran sepanjang waktu (dihitung, tidak disimpan)
	allIncome, allExpense, _, err := h.totals(ctx, bson.M{"user_id": uid})
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	mIncome, mExpense, mCount, err := h.totals(ctx, bson.M{
		"user_id":          uid,
		"transaction_date": bson.M{"$gte": p.Start, "$lt": p.End},
	})
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"period":            periodJSON(p, loc),
		"balance":           allIncome - allExpense,
		"month_income":      mIncome,
		"month_expense":     mExpense,
		"month_net":         mIncome - mExpense,
		"transaction_count": mCount,
	})
}

// Cashflow: GET /dashboard/cashflow?month=2026-10 -> satu titik per hari (hari kosong = 0)
func (h *DashboardHandler) Cashflow(c *gin.Context) {
	uid := middleware.UserID(c)
	ctx := c.Request.Context()
	loc := loadUserLocation(ctx, h.users, uid)

	p, err := parseMonth(c.Query("month"), loc)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "month harus berformat YYYY-MM, contoh 2026-10")
		return
	}

	cur, err := h.txs.Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{
			"user_id":          uid,
			"transaction_date": bson.M{"$gte": p.Start, "$lt": p.End},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				// Kelompokkan per hari menurut zona waktu user, bukan UTC
				"day": bson.M{"$dateToString": bson.M{
					"format":   "%Y-%m-%d",
					"date":     "$transaction_date",
					"timezone": loc.String(),
				}},
				"type": "$type",
			},
			"total": bson.M{"$sum": "$amount"},
		}}},
	})
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	var rows []struct {
		ID struct {
			Day  string `bson:"day"`
			Type string `bson:"type"`
		} `bson:"_id"`
		Total int64 `bson:"total"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	type dayTotals struct{ income, expense int64 }
	byDay := map[string]*dayTotals{}
	for _, r := range rows {
		d := byDay[r.ID.Day]
		if d == nil {
			d = &dayTotals{}
			byDay[r.ID.Day] = d
		}
		if r.ID.Type == models.TypeIncome {
			d.income = r.Total
		} else {
			d.expense = r.Total
		}
	}

	points := []gin.H{}
	for d := p.Start; d.Before(p.End); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		var inc, exp int64
		if t := byDay[key]; t != nil {
			inc, exp = t.income, t.expense
		}
		points = append(points, gin.H{"date": key, "income": inc, "expense": exp, "net": inc - exp})
	}

	c.JSON(http.StatusOK, gin.H{"period": periodJSON(p, loc), "data": points})
}

// Categories: GET /dashboard/categories?month=2026-10&type=expense
func (h *DashboardHandler) Categories(c *gin.Context) {
	uid := middleware.UserID(c)
	ctx := c.Request.Context()

	t := c.DefaultQuery("type", models.TypeExpense)
	if t != models.TypeIncome && t != models.TypeExpense {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "type harus income atau expense")
		return
	}
	loc := loadUserLocation(ctx, h.users, uid)
	p, err := parseMonth(c.Query("month"), loc)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "month harus berformat YYYY-MM, contoh 2026-10")
		return
	}

	cur, err := h.txs.Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{
			"user_id":          uid,
			"type":             t,
			"transaction_date": bson.M{"$gte": p.Start, "$lt": p.End},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   "$category_id",
			"total": bson.M{"$sum": "$amount"},
			"count": bson.M{"$sum": 1},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "total", Value: -1}}}},
	})
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	var rows []struct {
		CategoryID bson.ObjectID `bson:"_id"`
		Total      int64         `bson:"total"`
		Count      int64         `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	ids := bson.A{}
	var grand int64
	for _, r := range rows {
		ids = append(ids, r.CategoryID)
		grand += r.Total
	}
	cats := map[bson.ObjectID]models.Category{}
	if len(ids) > 0 {
		cc, err := h.categories.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
		if err != nil {
			httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
			return
		}
		var list []models.Category
		if err := cc.All(ctx, &list); err != nil {
			httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
			return
		}
		for _, cat := range list {
			cats[cat.ID] = cat
		}
	}

	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		pct := 0.0
		if grand > 0 {
			pct = math.Round(float64(r.Total)/float64(grand)*1000) / 10 // 1 desimal, hanya untuk tampilan
		}
		item := gin.H{
			"category_id": r.CategoryID,
			"name":        "Tanpa kategori",
			"icon":        "",
			"total":       r.Total,
			"count":       r.Count,
			"percent":     pct,
		}
		if cat, ok := cats[r.CategoryID]; ok {
			item["name"] = cat.Name
			item["icon"] = cat.Icon
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"period": periodJSON(p, loc),
		"type":   t,
		"total":  grand,
		"data":   items,
	})
}
