package domain

import (
	"time"
)

// SharedAnalysisLink rappresenta un link pubblico temporaneo per condividere un'analisi.
type SharedAnalysisLink struct {
	ID        string    `bson:"_id" json:"id"` // Token univoco
	DeckID    string    `bson:"deck_id" json:"deck_id"`
	UserID    string    `bson:"user_id,omitempty" json:"user_id,omitempty"`
	Channel   string    `bson:"channel" json:"channel"` // email, whatsapp, etc
	Recipient string    `bson:"recipient,omitempty" json:"recipient,omitempty"`
	Message   string    `bson:"message,omitempty" json:"message,omitempty"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"`
	VisitCount int      `bson:"visit_count" json:"visit_count"`
	LastVisit  time.Time `bson:"last_visit,omitempty" json:"last_visit,omitempty"`
}
