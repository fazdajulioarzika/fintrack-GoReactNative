package database

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func EnsureSavingsIndexes(ctx context.Context, db *mongo.Database) error {
	// Koleksi harus ada sebelum dipakai di dalam transaksi multi-dokumen.
	for _, name := range []string{"savings_goals", "savings_contributions"} {
		if err := db.CreateCollection(ctx, name); err != nil && !mongo.IsDuplicateKeyError(err) {
			// Jika koleksi sudah ada, Atlas mengembalikan error "NamespaceExists" (kode 48): abaikan.
			var cmdErr mongo.CommandError
			if !(asCommandError(err, &cmdErr) && cmdErr.Code == 48) {
				return err
			}
		}
	}

	_, err := db.Collection("savings_goals").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
	})
	if err != nil {
		return err
	}

	_, err = db.Collection("savings_contributions").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "goal_id", Value: 1}, {Key: "contribution_date", Value: -1}}},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "idempotency_key", Value: 1}},
			Options: options.Index().SetUnique(true).
				SetPartialFilterExpression(bson.M{"idempotency_key": bson.M{"$type": "string"}}),
		},
	})
	return err
}
func asCommandError(err error, target *mongo.CommandError) bool {
	return errors.As(err, target)
}
