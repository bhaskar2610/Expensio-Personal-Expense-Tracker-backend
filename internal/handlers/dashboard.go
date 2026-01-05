package handlers

import (
	"net/http"
	"time"

	"expensio/internal/db"
	"expensio/internal/models"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

type SummaryResponse struct {
	CurrentMonth   MonthSummary        `json:"current_month"`
	TripExpenses   TripExpensesSummary `json:"trip_expenses"`
	RecentExpenses []models.Expense    `json:"recent_expenses"`
}

type MonthSummary struct {
	Total float64 `json:"total"`
	Count int64   `json:"count"`
	Month string  `json:"month"`
}

type TripExpensesSummary struct {
	Total          float64 `json:"total"`
	TotalThisMonth float64 `json:"total_this_month"`
	Count          int64   `json:"count"`
	TripCount      int64   `json:"trip_count"`
}

type CategoryData struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}

func (h *DashboardHandler) GetSummary(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get month and year from query params, default to current month
	now := time.Now()
	month := c.DefaultQuery("month", now.Format("01"))
	year := c.DefaultQuery("year", now.Format("2006"))

	// Parse selected month/year
	startDate := year + "-" + month + "-01"
	selectedTime, _ := time.Parse("2006-01-02", startDate)
	monthName := selectedTime.Format("January 2006")

	// Calculate monthly total for selected month
	var total float64
	var count int64
	db.DB.Model(&models.Expense{}).
		Where("user_id = ?::uuid AND expense_date >= ?::date AND expense_date < (?::date + interval '1 month')", userID, startDate, startDate).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total)

	db.DB.Model(&models.Expense{}).
		Where("user_id = ?::uuid AND expense_date >= ?::date AND expense_date < (?::date + interval '1 month')", userID, startDate, startDate).
		Count(&count)

	// Get recent expenses for selected month
	recentExpenses := make([]models.Expense, 0)
	db.DB.Where("user_id = ?::uuid AND expense_date >= ?::date AND expense_date < (?::date + interval '1 month')", userID, startDate, startDate).
		Order("expense_date DESC, created_at DESC").
		Limit(5).
		Find(&recentExpenses)

	// Calculate trip expenses - ALL TIME (not just current month)
	var tripTotal float64
	var tripExpenseCount int64
	var tripCount int64

	// Get ALL expenses linked to trips (not filtered by month)
	db.DB.Raw(`
		SELECT COALESCE(SUM(e.amount), 0) as total, COUNT(DISTINCT e.id) as expense_count, COUNT(DISTINCT te.trip_id) as trip_count
		FROM expenses e
		INNER JOIN trip_expenses te ON e.id = te.expense_id
		WHERE e.user_id = ?::uuid
	`, userID).Row().Scan(&tripTotal, &tripExpenseCount, &tripCount)

	// Also get trip expenses for selected month only
	var tripTotalThisMonth float64
	db.DB.Raw(`
		SELECT COALESCE(SUM(e.amount), 0)
		FROM expenses e
		INNER JOIN trip_expenses te ON e.id = te.expense_id
		WHERE e.user_id = ?::uuid 
		AND e.expense_date >= ?::date 
		AND e.expense_date < (?::date + interval '1 month')
	`, userID, startDate, startDate).Row().Scan(&tripTotalThisMonth)

	response := SummaryResponse{
		CurrentMonth: MonthSummary{
			Total: total,
			Count: count,
			Month: monthName,
		},
		TripExpenses: TripExpensesSummary{
			Total:          tripTotal,
			TotalThisMonth: tripTotalThisMonth,
			Count:          tripExpenseCount,
			TripCount:      tripCount,
		},
		RecentExpenses: recentExpenses,
	}

	c.JSON(http.StatusOK, response)
}

func (h *DashboardHandler) GetPieChart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get month and year from query params, default to current month
	month := c.DefaultQuery("month", time.Now().Format("01"))
	year := c.DefaultQuery("year", time.Now().Format("2006"))

	startDate := year + "-" + month + "-01"

	categoryData := make([]CategoryData, 0) // Initialize as empty slice, not nil
	db.DB.Model(&models.Expense{}).
		Select("category, SUM(amount) as amount").
		Where("user_id = ?::uuid AND expense_date >= ?::date AND expense_date < (?::date + interval '1 month')", userID, startDate, startDate).
		Group("category").
		Order("amount DESC").
		Scan(&categoryData)

	c.JSON(http.StatusOK, categoryData)
}
