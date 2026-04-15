package domain

import "time"

// PasswordResetToken is a one-time token used to reset account passwords.
type PasswordResetToken struct {
	Token     string     `bson:"_id" json:"token"`
	UserID    string     `bson:"user_id" json:"user_id"`
	Email     string     `bson:"email" json:"email"`
	Purpose   string     `bson:"purpose" json:"purpose"`
	ExpiresAt time.Time  `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UsedAt    *time.Time `bson:"used_at,omitempty" json:"used_at,omitempty"`
}
