package domain

import "time"

// Plan represents a user's subscription plan.
type Plan string

const (
	PlanFree Plan = "free"
	PlanPro  Plan = "pro"
)

// User represents an application user.
type User struct {
	ID              string    `bson:"_id"              json:"id"`
	Email           string    `bson:"email"            json:"email"`
	PasswordHash    string    `bson:"password_hash"    json:"-"`
	Name            string    `bson:"name"             json:"name"`
	Plan            Plan      `bson:"plan"             json:"plan"`
	Remaining       int       `bson:"-"                json:"remaining,omitempty"`
	DiscordID       string    `bson:"discord_id,omitempty" json:"discord_id,omitempty"`
	DailyAnalyses   int       `bson:"daily_analyses"   json:"daily_analyses"`
	LastAnalysisDay string    `bson:"last_analysis_day" json:"-"` // format: "2006-01-02"
	CreatedAt       time.Time `bson:"created_at"       json:"created_at"`
	UpdatedAt       time.Time `bson:"updated_at"       json:"updated_at"`
}

// FreeDailyLimit is the number of analyses allowed per day for free users.
const FreeDailyLimit = 3

// CanAnalyze returns true if the user has remaining quota for today.
func (u *User) CanAnalyze(today string) bool {
	if u.Plan == PlanPro {
		return true
	}
	if u.LastAnalysisDay != today {
		return true
	}
	return u.DailyAnalyses < FreeDailyLimit
}

// RemainingAnalyses returns the number of analyses left today for free users.
// For pro users, it returns -1 (unlimited).
func (u *User) RemainingAnalyses(today string) int {
	if u.Plan == PlanPro {
		return -1
	}
	if u.LastAnalysisDay != today {
		return FreeDailyLimit
	}
	remaining := FreeDailyLimit - u.DailyAnalyses
	if remaining < 0 {
		return 0
	}
	return remaining
}
