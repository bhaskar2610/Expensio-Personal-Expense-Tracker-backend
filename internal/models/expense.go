package models

import (
	"time"

	"github.com/google/uuid"
)

type Expense struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Title       string    `gorm:"not null" json:"title"`
	Category    string    `gorm:"not null" json:"category"`
	Amount      float64   `gorm:"type:numeric;not null" json:"amount"`
	Note        string    `json:"note"`
	ExpenseDate time.Time `gorm:"type:date;not null" json:"expense_date"`
	CreatedAt   time.Time `gorm:"default:now()" json:"created_at"`
	User        User      `gorm:"foreignKey:UserID" json:"-"`
}
