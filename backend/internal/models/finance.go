package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	TypeIncome  = "income"
	TypeExpense = "expense"
)

// Category: UserID nil = kategori sistem (default), terisi = kategori milik user (fitur lanjutan).
type Category struct {
	ID        bson.ObjectID  `bson:"_id" json:"id"`
	UserID    *bson.ObjectID `bson:"user_id" json:"-"`
	Name      string         `bson:"name" json:"name"`
	Type      string         `bson:"type" json:"type"`
	Icon      string         `bson:"icon" json:"icon"`
	IsSystem  bool           `bson:"is_system" json:"is_system"`
	SortOrder int            `bson:"sort_order" json:"-"`
	CreatedAt time.Time      `bson:"created_at" json:"created_at"`
}

type Transaction struct {
	ID              bson.ObjectID `bson:"_id"`
	UserID          bson.ObjectID `bson:"user_id"`
	Type            string        `bson:"type"`
	Amount          int64         `bson:"amount"` // rupiah utuh, tanpa desimal
	CategoryID      bson.ObjectID `bson:"category_id"`
	Note            string        `bson:"note"`
	PaymentMethod   string        `bson:"payment_method"`
	TransactionDate time.Time     `bson:"transaction_date"`
	IdempotencyKey  string        `bson:"idempotency_key,omitempty"`
	CreatedAt       time.Time     `bson:"created_at"`
	UpdatedAt       time.Time     `bson:"updated_at"`
}