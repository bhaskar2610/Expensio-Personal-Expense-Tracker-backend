package models

import (
	"time"

	"github.com/google/uuid"
)

type Trip struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Name      string    `gorm:"not null" json:"name"`
	StartDate time.Time `gorm:"type:date" json:"start_date"`
	EndDate   time.Time `gorm:"type:date" json:"end_date"`
	CreatedAt time.Time `gorm:"default:now()" json:"created_at"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
	Expenses  []Expense `gorm:"many2many:trip_expenses;" json:"expenses,omitempty"`
}

type TripExpense struct {
	TripID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"trip_id"`
	ExpenseID uuid.UUID `gorm:"type:uuid;primaryKey" json:"expense_id"`
}
