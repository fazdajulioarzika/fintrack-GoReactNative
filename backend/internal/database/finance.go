package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type defaultCategory struct {
	Name string
	Type string
	Icon string // nama ikon Ionicons (dipakai di mobile nanti)
}

var defaultCategories = []defaultCategory{
	{"Makanan dan minuman", "expense", "restaurant"},
	{"Transportasi", "expense", "car"},
	{"Belanja", "expense", "bag"},
	{"Kesehatan", "expense", "medkit"},
	{"Pendidikan", "expense", "school"},
	{"Hiburan", "expense", "game-controller"},
	{"Tagihan", "expense", "receipt"},
	{"Lainnya", "expense", "ellipsis-horizontal"},

	{"Gaji", "income", "cash"},
	{"Freelance", "income", "laptop"},
	{"Bisnis", "income", "briefcase"},
	{"Bonus", "income", "gift"},
	{"Investasi", "income", "trending-up"},
	{"Lainnya", "income", "ellipsis-horizontal"},
}

// EnsureFinanceIndexes membuat indeks untuk transaksi dan kategori (PRD 10.7).
func EnsureFinanceIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("transactions").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "transaction_date", Value: -1}}},
		{Keys: bson.D{
			{Key: "user_id", Value: 1}, {Key: "type", Value: 1},
			{Key: "category_id", Value: 1}, {Key: "transaction_date", Value: -1},
		}},
		{
			// Unik per user, tapi hanya untuk dokumen yang punya idempotency_key
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "idempotency_key", Value: 1}},
			Options: options.Index().SetUnique(true).
				SetPartialFilterExpression(bson.M{"idempotency_key": bson.M{"$type": "string"}}),
		},
	})
	if err != nil {
		return err
	}

	_, err = db.Collection("categories").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "type", Value: 1}, {Key: "sort_order", Value: 1}},
	})
	return err
}

// SeedCategories memasukkan kategori default. Aman dipanggil berulang (upsert).
func SeedCategories(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection("categories")
	now := time.Now().UTC()

	for i, c := range defaultCategories {
		_, err := coll.UpdateOne(ctx,
			bson.M{"user_id": nil, "name": c.Name, "type": c.Type},
			bson.M{"$setOnInsert": bson.M{
				"_id":        bson.NewObjectID(),
				"icon":       c.Icon,
				"is_system":  true,
				"sort_order": i,
				"created_at": now,
			}},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}
	return nil
}