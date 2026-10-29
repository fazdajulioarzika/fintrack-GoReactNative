// Package models berisi struct yang dipetakan ke dokumen MongoDB.
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Tag `bson` = nama field di MongoDB, tag `json` = nama field di respons API.
// json:"-" artinya field TIDAK PERNAH dikirim ke klien.
type User struct {
	ID           bson.ObjectID `bson:"_id" json:"id"`
	Name         string        `bson:"name" json:"name"`
	Email        string        `bson:"email" json:"email"`
	PasswordHash string        `bson:"password_hash" json:"-"`
	Currency     string        `bson:"currency" json:"currency"`
	Timezone     string        `bson:"timezone" json:"timezone"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at" json:"updated_at"`
}

// Session menyimpan HASH refresh token (bukan tokennya) agar logout bisa mencabut sesi.
type Session struct {
	ID        bson.ObjectID `bson:"_id"`
	UserID    bson.ObjectID `bson:"user_id"`
	TokenHash string        `bson:"token_hash"`
	UserAgent string        `bson:"user_agent"`
	ExpiresAt time.Time     `bson:"expires_at"`
	CreatedAt time.Time     `bson:"created_at"`
}
