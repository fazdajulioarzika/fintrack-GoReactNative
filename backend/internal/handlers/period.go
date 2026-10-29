package handlers

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// period = satu bulan kalender menurut zona waktu user. Start inklusif, End eksklusif.
type period struct {
	Label string // "2026-10"
	Start time.Time
	End   time.Time
}

// parseMonth membaca "YYYY-MM". String kosong = bulan berjalan menurut zona waktu user.
func parseMonth(s string, loc *time.Location) (period, error) {
	var y int
	var m time.Month
	if s == "" {
		now := time.Now().In(loc)
		y, m = now.Year(), now.Month()
	} else {
		t, err := time.ParseInLocation("2006-01", s, loc)
		if err != nil {
			return period{}, err
		}
		y, m = t.Year(), t.Month()
	}
	start := time.Date(y, m, 1, 0, 0, 0, 0, loc)
	return period{Label: start.Format("2006-01"), Start: start, End: start.AddDate(0, 1, 0)}, nil
}

func periodJSON(p period, loc *time.Location) gin.H {
	return gin.H{
		"month":    p.Label,
		"start":    p.Start,
		"end":      p.End,
		"timezone": loc.String(),
	}
}

// loadUserLocation mengambil zona waktu user; fallback ke Asia/Jakarta.
func loadUserLocation(ctx context.Context, users *mongo.Collection, uid bson.ObjectID) *time.Location {
	var u struct {
		Timezone string `bson:"timezone"`
	}
	err := users.FindOne(ctx, bson.M{"_id": uid},
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
