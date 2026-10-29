package database

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EnsureBudgetIndexes: satu anggaran per (user, kategori, bulan) dijamin oleh unique index.
func EnsureBudgetIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection("budgets").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1}, {Key: "category_id", Value: 1}, {Key: "period", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "period", Value: 1}}},
	})
	return err
}
