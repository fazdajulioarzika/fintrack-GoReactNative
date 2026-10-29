package handlers

import (
	"context"
	"errors"
	"math"
	"net/http"
	"time"

	"fintrack-backend/internal/httpx"
	"fintrack-backend/internal/middleware"
	"fintrack-backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type AnalyticsHandler struct {
	txs        *mongo.Collection
	categories *mongo.Collection
	users      *mongo.Collection
}

func NewAnalyticsHandler(db *mongo.Database) *AnalyticsHandler {
	return &AnalyticsHandler{
		txs:        db.Collection("transactions"),
		categories: db.Collection("categories"),
		users:      db.Collection("users"),
	}
}

// dateRange: Start inklusif, End eksklusif (sudah dalam zona waktu user).
type dateRange struct {
	Label string
	Start time.Time
	End   time.Time
}

func (r dateRange) json() gin.H {
	return gin.H{
		"label":      r.Label,
		"start_date": r.Start.Format("2006-01-02"),
		"end_date":   r.End.AddDate(0, 0, -1).Format("2006-01-02"), // tampilkan inklusif
	}
}

func (r dateRange) match(uid bson.ObjectID) bson.M {
	return bson.M{"user_id": uid, "transaction_date": bson.M{"$gte": r.Start, "$lt": r.End}}
}

func monthRange(first time.Time) dateRange {
	return dateRange{Label: first.Format("2006-01"), Start: first, End: first.AddDate(0, 1, 0)}
}

// resolveRange membaca ?range=this_month|last_month|custom (&start_date=&end_date=) dan
// mengembalikan periode saat ini beserta periode pembandingnya.
func resolveRange(c *gin.Context, loc *time.Location) (cur, prev dateRange, err error) {
	switch c.DefaultQuery("range", "this_month") {
	case "this_month", "last_month":
		now := time.Now().In(loc)
		base := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		if c.Query("range") == "last_month" {
			base = base.AddDate(0, -1, 0)
		}
		return monthRange(base), monthRange(base.AddDate(0, -1, 0)), nil

	case "custom":
		start, e1 := time.ParseInLocation("2006-01-02", c.Query("start_date"), loc)
		end, e2 := time.ParseInLocation("2006-01-02", c.Query("end_date"), loc)
		if e1 != nil || e2 != nil {
			return cur, prev, errors.New("range=custom membutuhkan start_date dan end_date (YYYY-MM-DD)")
		}
		if end.Before(start) {
			return cur, prev, errors.New("end_date tidak boleh sebelum start_date")
		}
		endEx := end.AddDate(0, 0, 1)
		days := int(math.Round(endEx.Sub(start).Hours() / 24))
		if days > 366 {
			return cur, prev, errors.New("rentang tanggal maksimal 366 hari")
		}
		cur = dateRange{Label: "custom", Start: start, End: endEx}
		prev = dateRange{Label: "previous", Start: start.AddDate(0, 0, -days), End: start}
		return cur, prev, nil
	}
	return cur, prev, errors.New("range harus this_month, last_month, atau custom")
}

// ---------- Handlers ----------

// Summary: GET /analytics/summary -> total, selisih, dan distribusi pengeluaran per kategori
func (h *AnalyticsHandler) Summary(c *gin.Context) {
	uid := middleware.UserID(c)
	ctx := c.Request.Context()
	loc := loadUserLocation(ctx, h.users, uid)

	cur, _, err := resolveRange(c, loc)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	income, expense, count, err := aggregateTotals(ctx, h.txs, cur.match(uid))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	cats, catTotal, err := h.categoryBreakdown(ctx, uid, models.TypeExpense, cur)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"period":                    cur.json(),
		"timezone":                  loc.String(),
		"total_income":              income,
		"total_expense":             expense,
		"net":                       income - expense,
		"transaction_count":         count,
		"expense_by_category":       cats,
		"expense_by_category_total": catTotal,
	})
}

// Trend: GET /analytics/trend?granularity=day|month
func (h *AnalyticsHandler) Trend(c *gin.Context) {
	uid := middleware.UserID(c)
	ctx := c.Request.Context()
	loc := loadUserLocation(ctx, h.users, uid)

	cur, _, err := resolveRange(c, loc)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	gran := c.DefaultQuery("granularity", "day")
	if gran != "day" && gran != "month" {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "granularity harus day atau month")
		return
	}
	format, layout := "%Y-%m-%d", "2006-01-02"
	if gran == "month" {
		format, layout = "%Y-%m", "2006-01"
	}

	agg, err := h.txs.Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$match", Value: cur.match(uid)}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"bucket": bson.M{"$dateToString": bson.M{
					"format": format, "date": "$transaction_date", "timezone": loc.String(),
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
			Bucket string `bson:"bucket"`
			Type   string `bson:"type"`
		} `bson:"_id"`
		Total int64 `bson:"total"`
	}
	if err := agg.All(ctx, &rows); err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	type pair struct{ income, expense int64 }
	byBucket := map[string]*pair{}
	for _, r := range rows {
		p := byBucket[r.ID.Bucket]
		if p == nil {
			p = &pair{}
			byBucket[r.ID.Bucket] = p
		}
		if r.ID.Type == models.TypeIncome {
			p.income = r.Total
		} else {
			p.expense = r.Total
		}
	}

	// Isi bucket kosong dengan 0 supaya grafik tidak bolong
	points := []gin.H{}
	step := func(t time.Time) time.Time { return t.AddDate(0, 0, 1) }
	start := cur.Start
	if gran == "month" {
		step = func(t time.Time) time.Time { return t.AddDate(0, 1, 0) }
		start = time.Date(cur.Start.Year(), cur.Start.Month(), 1, 0, 0, 0, 0, loc)
	}
	for d := start; d.Before(cur.End); d = step(d) {
		key := d.Format(layout)
		var inc, exp int64
		if p := byBucket[key]; p != nil {
			inc, exp = p.income, p.expense
		}
		points = append(points, gin.H{"date": key, "income": inc, "expense": exp, "net": inc - exp})
	}

	c.JSON(http.StatusOK, gin.H{
		"period":      cur.json(),
		"granularity": gran,
		"data":        points,
	})
}

// Compare: GET /analytics/compare -> periode ini vs periode sebelumnya
func (h *AnalyticsHandler) Compare(c *gin.Context) {
	uid := middleware.UserID(c)
	ctx := c.Request.Context()
	loc := loadUserLocation(ctx, h.users, uid)

	cur, prev, err := resolveRange(c, loc)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	ci, ce, cn, err := aggregateTotals(ctx, h.txs, cur.match(uid))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	pi, pe, pn, err := aggregateTotals(ctx, h.txs, prev.match(uid))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current":  gin.H{"period": cur.json(), "income": ci, "expense": ce, "net": ci - ce, "transaction_count": cn},
		"previous": gin.H{"period": prev.json(), "income": pi, "expense": pe, "net": pi - pe, "transaction_count": pn},
		// percent bernilai null jika periode pembanding 0 (tidak bisa dibagi)
		"change": gin.H{
			"income":  changeJSON(ci, pi),
			"expense": changeJSON(ce, pe),
			"net":     changeJSON(ci-ce, pi-pe),
		},
	})
}

// ---------- Helper ----------

func (h *AnalyticsHandler) categoryBreakdown(ctx context.Context, uid bson.ObjectID, txType string, r dateRange) ([]gin.H, int64, error) {
	match := r.match(uid)
	match["type"] = txType

	agg, err := h.txs.Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":   "$category_id",
			"total": bson.M{"$sum": "$amount"},
			"count": bson.M{"$sum": 1},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "total", Value: -1}}}},
	})
	if err != nil {
		return nil, 0, err
	}
	var rows []struct {
		CategoryID bson.ObjectID `bson:"_id"`
		Total      int64         `bson:"total"`
		Count      int64         `bson:"count"`
	}
	if err := agg.All(ctx, &rows); err != nil {
		return nil, 0, err
	}

	ids := bson.A{}
	var grand int64
	for _, row := range rows {
		ids = append(ids, row.CategoryID)
		grand += row.Total
	}
	catByID := map[bson.ObjectID]models.Category{}
	if len(ids) > 0 {
		cc, err := h.categories.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
		if err != nil {
			return nil, 0, err
		}
		var list []models.Category
		if err := cc.All(ctx, &list); err != nil {
			return nil, 0, err
		}
		for _, cat := range list {
			catByID[cat.ID] = cat
		}
	}

	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		pct := 0.0
		if grand > 0 {
			pct = math.Round(float64(row.Total)/float64(grand)*1000) / 10
		}
		item := gin.H{
			"category_id": row.CategoryID, "name": "Tanpa kategori", "icon": "",
			"total": row.Total, "count": row.Count, "percent": pct,
		}
		if cat, ok := catByID[row.CategoryID]; ok {
			item["name"], item["icon"] = cat.Name, cat.Icon
		}
		items = append(items, item)
	}
	return items, grand, nil
}

// aggregateTotals menjumlahkan pemasukan dan pengeluaran untuk filter tertentu.
func aggregateTotals(ctx context.Context, txs *mongo.Collection, match bson.M) (income, expense, count int64, err error) {
	cur, err := txs.Aggregate(ctx, mongo.Pipeline{
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

// changePercent: perubahan relatif terhadap periode sebelumnya; nil jika pembanding 0.
func changePercent(cur, prev int64) *float64 {
	if prev == 0 {
		return nil
	}
	v := math.Round(float64(cur-prev)/math.Abs(float64(prev))*1000) / 10
	return &v
}

func changeJSON(cur, prev int64) gin.H {
	return gin.H{"amount": cur - prev, "percent": changePercent(cur, prev)}
}
