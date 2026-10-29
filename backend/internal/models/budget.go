package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Budget: batas pengeluaran per kategori per bulan.
// Period berformat "YYYY-MM". Jumlah terpakai TIDAK disimpan, selalu dihitung dari transaksi.
type Budget struct {
	ID          bson.ObjectID `bson:"_id"`
	UserID      bson.ObjectID `bson:"user_id"`
	CategoryID  bson.ObjectID `bson:"category_id"`
	Period      string        `bson:"period"`
	LimitAmount int64         `bson:"limit_amount"`
	CreatedAt   time.Time     `bson:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at"`
}
