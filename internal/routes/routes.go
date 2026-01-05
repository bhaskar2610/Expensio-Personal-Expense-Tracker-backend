package routes

import (
	"expensio/internal/config"
	"expensio/internal/handlers"
	"expensio/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	r := gin.Default()

	// CORS middleware - Must be before routes
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:5174", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(cfg)
	expenseHandler := handlers.NewExpenseHandler()
	tripHandler := handlers.NewTripHandler()
	dashboardHandler := handlers.NewDashboardHandler()

	// Public routes
	auth := r.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Protected routes
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(cfg))
	{
		// Expenses
		expenses := protected.Group("/expenses")
		{
			expenses.GET("", expenseHandler.GetExpenses)
			expenses.POST("", expenseHandler.CreateExpense)
			expenses.PUT("/:id", expenseHandler.UpdateExpense)
			expenses.DELETE("/:id", expenseHandler.DeleteExpense)
		}

		// Trips
		trips := protected.Group("/trips")
		{
			trips.GET("", tripHandler.GetTrips)
			trips.POST("", tripHandler.CreateTrip)
			trips.GET("/:id", tripHandler.GetTrip)
			trips.DELETE("/:id", tripHandler.DeleteTrip)
			trips.POST("/:id/expenses", tripHandler.AddExpenseToTrip)
		}

		// Dashboard
		dashboard := protected.Group("/dashboard")
		{
			dashboard.GET("/summary", dashboardHandler.GetSummary)
			dashboard.GET("/pie-chart", dashboardHandler.GetPieChart)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return r
}
