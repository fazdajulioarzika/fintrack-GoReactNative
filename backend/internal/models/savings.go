package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	GoalActive      = "active"
	GoalAchieved    = "achieved"
	ContribDeposit  = "deposit"
	ContribWithdraw = "withdraw"
)

type SavingsGoal struct {
	ID           bson.ObjectID `bson:"_id"`
	UserID       bson.ObjectID `bson:"user_id"`
	Name         string        `bson:"name"`
	TargetAmount int64         `bson:"target_amount"`
	SavedAmount  int64         `bson:"saved_amount"`
	TargetDate   *time.Time    `bson:"target_date,omitempty"`
	Icon         string        `bson:"icon"`
	Status       string        `bson:"status"`
	CreatedAt    time.Time     `bson:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at"`
}

// SavingsContribution: riwayat setoran/penarikan. Amount selalu positif, arahnya ditentukan Type.
type SavingsContribution struct {
	ID               bson.ObjectID `bson:"_id"`
	UserID           bson.ObjectID `bson:"user_id"`
	GoalID           bson.ObjectID `bson:"goal_id"`
	Type             string        `bson:"type"`
	Amount           int64         `bson:"amount"`
	ContributionDate time.Time     `bson:"contribution_date"`
	Note             string        `bson:"note"`
	IdempotencyKey   string        `bson:"idempotency_key,omitempty"`
	CreatedAt        time.Time     `bson:"created_at"`
}
