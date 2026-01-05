package handlers

import (
	"net/http"
	"time"

	"expensio/internal/db"
	"expensio/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ExpenseHandler struct{}

func NewExpenseHandler() *ExpenseHandler {
	return &ExpenseHandler{}
}

type ExpenseResponse struct {
	models.Expense
	TripID   *string `json:"trip_id,omitempty"`
	TripName *string `json:"trip_name,omitempty"`
}

type CreateExpenseRequest struct {
	Title       string  `json:"title" binding:"required"`
	Category    string  `json:"category" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Note        string  `json:"note"`
	ExpenseDate string  `json:"expense_date" binding:"required"`
	TripID      *string `json:"trip_id"` // Optional trip ID
}

type UpdateExpenseRequest struct {
	Title       string  `json:"title"`
	Category    string  `json:"category"`
	Amount      float64 `json:"amount" binding:"omitempty,gt=0"`
	Note        string  `json:"note"`
	ExpenseDate string  `json:"expense_date"`
	TripID      *string `json:"trip_id"` // Optional trip ID (use empty string to unlink)
}

func (h *ExpenseHandler) GetExpenses(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse query params for filtering
	month := c.Query("month")
	year := c.Query("year")

	query := db.DB.Where("user_id = ?::uuid", userID)

	if month != "" && year != "" {
		startDate := year + "-" + month + "-01"
		query = query.Where("expense_date >= ?::date AND expense_date < (?::date + interval '1 month')", startDate, startDate)
	}

	var expenses []models.Expense
	if err := query.Order("expense_date DESC").Find(&expenses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch expenses"})
		return
	}

	// Fetch trip information for each expense
	response := make([]ExpenseResponse, 0) // Initialize as empty slice, not nil
	for _, expense := range expenses {
		expenseResp := ExpenseResponse{
			Expense: expense,
		}

		// Get trip info if linked
		var tripExpense models.TripExpense
		if err := db.DB.Where("expense_id = ?", expense.ID).First(&tripExpense).Error; err == nil {
			var trip models.Trip
			if err := db.DB.Where("id = ?", tripExpense.TripID).First(&trip).Error; err == nil {
				tripIDStr := trip.ID.String()
				expenseResp.TripID = &tripIDStr
				expenseResp.TripName = &trip.Name
			}
		}

		response = append(response, expenseResp)
	}

	c.JSON(http.StatusOK, response)
}

func (h *ExpenseHandler) CreateExpense(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	expenseDate, err := time.Parse("2006-01-02", req.ExpenseDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	expense := models.Expense{
		UserID:      userUUID,
		Title:       req.Title,
		Category:    req.Category,
		Amount:      req.Amount,
		Note:        req.Note,
		ExpenseDate: expenseDate,
	}

	if err := db.DB.Create(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create expense"})
		return
	}

	// Link to trip if trip_id is provided
	if req.TripID != nil && *req.TripID != "" {
		tripID := *req.TripID
		// Verify trip exists and belongs to user
		var trip models.Trip
		if err := db.DB.Where("id = ? AND user_id = ?::uuid", tripID, userID).First(&trip).Error; err == nil {
			// Link expense to trip
			db.DB.Model(&trip).Association("Expenses").Append(&expense)
		}
	}

	c.JSON(http.StatusCreated, expense)
}

func (h *ExpenseHandler) UpdateExpense(c *gin.Context) {
	userID := c.GetString("user_id")
	expenseID := c.Param("id")

	var expense models.Expense
	if err := db.DB.Where("id = ? AND user_id = ?", expenseID, userID).First(&expense).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found"})
		return
	}

	var req UpdateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Title != "" {
		expense.Title = req.Title
	}
	if req.Category != "" {
		expense.Category = req.Category
	}
	if req.Amount > 0 {
		expense.Amount = req.Amount
	}
	expense.Note = req.Note
	if req.ExpenseDate != "" {
		expenseDate, err := time.Parse("2006-01-02", req.ExpenseDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
			return
		}
		expense.ExpenseDate = expenseDate
	}

	// Handle trip linking/unlinking
	if req.TripID != nil {
		tripID := *req.TripID
		if tripID == "" {
			// Unlink from all trips
			db.DB.Exec("DELETE FROM trip_expenses WHERE expense_id = ?", expenseID)
		} else {
			// Verify trip exists and belongs to user
			var trip models.Trip
			if err := db.DB.Where("id = ? AND user_id = ?::uuid", tripID, userID).First(&trip).Error; err == nil {
				// Clear existing trip links
				db.DB.Exec("DELETE FROM trip_expenses WHERE expense_id = ?", expenseID)
				// Link to new trip
				db.DB.Model(&trip).Association("Expenses").Append(&expense)
			}
		}
	}

	if err := db.DB.Save(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update expense"})
		return
	}

	c.JSON(http.StatusOK, expense)
}

func (h *ExpenseHandler) DeleteExpense(c *gin.Context) {
	userID := c.GetString("user_id")
	expenseID := c.Param("id")

	result := db.DB.Where("id = ? AND user_id = ?", expenseID, userID).Delete(&models.Expense{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete expense"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expense deleted successfully"})
}
