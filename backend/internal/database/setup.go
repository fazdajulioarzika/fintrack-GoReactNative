package database

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Setup menjalankan seluruh pembuatan indeks dan seed data. Aman dipanggil berulang.
func Setup(ctx context.Context, db *mongo.Database) error {
	steps := []func(context.Context, *mongo.Database) error{
		EnsureIndexes,
		EnsureFinanceIndexes,
		SeedCategories,
		EnsureBudgetIndexes,
		EnsureSavingsIndexes,
	}
	for _, step := range steps {
		if err := step(ctx, db); err != nil {
			return err
		}
	}
	return nil
}
