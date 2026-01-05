package handlers

import (
	"net/http"
	"time"

	"expensio/internal/db"
	"expensio/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TripHandler struct{}

func NewTripHandler() *TripHandler {
	return &TripHandler{}
}

type CreateTripRequest struct {
	Name      string `json:"name" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

type TripResponse struct {
	models.Trip
	TotalAmount float64 `json:"total_amount"`
}

func (h *TripHandler) GetTrips(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	trips := make([]models.Trip, 0) // Initialize as empty slice, not nil
	if err := db.DB.Where("user_id = ?::uuid", userID).Order("start_date DESC").Find(&trips).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch trips"})
		return
	}

	c.JSON(http.StatusOK, trips)
}

func (h *TripHandler) CreateTrip(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateTripRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format. Use YYYY-MM-DD"})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Use YYYY-MM-DD"})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	trip := models.Trip{
		UserID:    userUUID,
		Name:      req.Name,
		StartDate: startDate,
		EndDate:   endDate,
	}

	if err := db.DB.Create(&trip).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create trip"})
		return
	}

	c.JSON(http.StatusCreated, trip)
}

func (h *TripHandler) GetTrip(c *gin.Context) {
	userID := c.GetString("user_id")
	tripID := c.Param("id")

	var trip models.Trip
	if err := db.DB.Preload("Expenses").Where("id = ? AND user_id = ?::uuid", tripID, userID).First(&trip).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trip not found"})
		return
	}

	// Calculate total amount
	var totalAmount float64
	for _, expense := range trip.Expenses {
		totalAmount += expense.Amount
	}

	response := TripResponse{
		Trip:        trip,
		TotalAmount: totalAmount,
	}

	c.JSON(http.StatusOK, response)
}

type AddExpenseToTripRequest struct {
	ExpenseID string `json:"expense_id" binding:"required"`
}

func (h *TripHandler) AddExpenseToTrip(c *gin.Context) {
	userID := c.GetString("user_id")
	tripID := c.Param("id")

	var req AddExpenseToTripRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify trip exists and belongs to user
	var trip models.Trip
	if err := db.DB.Where("id = ? AND user_id = ?::uuid", tripID, userID).First(&trip).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trip not found"})
		return
	}

	// Verify expense exists and belongs to user
	var expense models.Expense
	if err := db.DB.Where("id = ? AND user_id = ?::uuid", req.ExpenseID, userID).First(&expense).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found"})
		return
	}

	// Add expense to trip
	if err := db.DB.Model(&trip).Association("Expenses").Append(&expense); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add expense to trip"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expense added to trip successfully"})
}

// DeleteTrip deletes a trip and optionally all its linked expenses
func (h *TripHandler) DeleteTrip(c *gin.Context) {
	userID := c.GetString("user_id")
	tripID := c.Param("id")

	// Check if we should also delete linked expenses (default: true)
	deleteExpenses := c.DefaultQuery("delete_expenses", "true") == "true"

	// Verify trip exists and belongs to user
	var trip models.Trip
	if err := db.DB.Where("id = ? AND user_id = ?::uuid", tripID, userID).First(&trip).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trip not found"})
		return
	}

	// Get linked expense IDs before deletion
	var expenseIDs []uuid.UUID
	if deleteExpenses {
		db.DB.Raw(`
			SELECT expense_id FROM trip_expenses WHERE trip_id = ?
		`, tripID).Scan(&expenseIDs)
	}

	// Delete trip_expenses links first
	db.DB.Exec("DELETE FROM trip_expenses WHERE trip_id = ?", tripID)

	// Delete the linked expenses if requested
	if deleteExpenses && len(expenseIDs) > 0 {
		db.DB.Where("id IN ?", expenseIDs).Delete(&models.Expense{})
	}

	// Delete the trip
	if err := db.DB.Delete(&trip).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete trip"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Trip deleted successfully",
		"expenses_deleted": len(expenseIDs),
	})
}
