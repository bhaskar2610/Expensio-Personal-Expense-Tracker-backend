package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"expensio/internal/db"
	"expensio/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type ReportHandler struct{}

func NewReportHandler() *ReportHandler {
	return &ReportHandler{}
}

type ExpenseRow struct {
	Date     string
	Title    string
	Category string
	Amount   float64
	Note     string
	Trip     string
}

type CategorySummary struct {
	Category string
	Total    float64
	Count    int
}

func (h *ReportHandler) ExportMonthlyReport(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get month and year from query params
	now := time.Now()
	month := c.DefaultQuery("month", now.Format("01"))
	year := c.DefaultQuery("year", now.Format("2006"))

	startDate := year + "-" + month + "-01"
	selectedTime, _ := time.Parse("2006-01-02", startDate)
	monthName := selectedTime.Format("January 2006")

	// Fetch expenses for the month
	var expenses []models.Expense
	db.DB.Where("user_id = ?::uuid AND expense_date >= ?::date AND expense_date < (?::date + interval '1 month')", userID, startDate, startDate).
		Order("expense_date ASC, created_at ASC").
		Find(&expenses)

	// Fetch trip info for each expense
	expenseRows := make([]ExpenseRow, 0)
	categoryMap := make(map[string]*CategorySummary)

	for _, expense := range expenses {
		tripName := ""
		var tripExpense models.TripExpense
		if err := db.DB.Where("expense_id = ?", expense.ID).First(&tripExpense).Error; err == nil {
			var trip models.Trip
			if err := db.DB.Where("id = ?", tripExpense.TripID).First(&trip).Error; err == nil {
				tripName = trip.Name
			}
		}

		expenseRows = append(expenseRows, ExpenseRow{
			Date:     expense.ExpenseDate.Format("02-Jan-2006"),
			Title:    expense.Title,
			Category: expense.Category,
			Amount:   expense.Amount,
			Note:     expense.Note,
			Trip:     tripName,
		})

		// Update category summary
		if _, exists := categoryMap[expense.Category]; !exists {
			categoryMap[expense.Category] = &CategorySummary{
				Category: expense.Category,
				Total:    0,
				Count:    0,
			}
		}
		categoryMap[expense.Category].Total += expense.Amount
		categoryMap[expense.Category].Count++
	}

	// Convert map to slice and sort by total
	categorySummaries := make([]CategorySummary, 0)
	for _, summary := range categoryMap {
		categorySummaries = append(categorySummaries, *summary)
	}
	sort.Slice(categorySummaries, func(i, j int) bool {
		return categorySummaries[i].Total > categorySummaries[j].Total
	})

	// Create Excel file
	f := excelize.NewFile()
	defer f.Close()

	// ===== EXPENSES SHEET =====
	expenseSheet := "Expenses"
	f.SetSheetName("Sheet1", expenseSheet)

	// Header styling
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"14B8A6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Title row
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "14B8A6"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	f.MergeCell(expenseSheet, "A1", "F1")
	f.SetCellValue(expenseSheet, "A1", fmt.Sprintf("Expense Report - %s", monthName))
	f.SetCellStyle(expenseSheet, "A1", "F1", titleStyle)
	f.SetRowHeight(expenseSheet, 1, 30)

	// Headers
	headers := []string{"Date", "Title", "Category", "Amount (₹)", "Note", "Trip"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c3", 'A'+i)
		f.SetCellValue(expenseSheet, cell, header)
		f.SetCellStyle(expenseSheet, cell, cell, headerStyle)
	}

	// Data rows
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
		},
	})

	amountStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 4, // Number with 2 decimal places
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
		},
	})

	var totalAmount float64
	for i, row := range expenseRows {
		rowNum := i + 4
		f.SetCellValue(expenseSheet, fmt.Sprintf("A%d", rowNum), row.Date)
		f.SetCellValue(expenseSheet, fmt.Sprintf("B%d", rowNum), row.Title)
		f.SetCellValue(expenseSheet, fmt.Sprintf("C%d", rowNum), row.Category)
		f.SetCellValue(expenseSheet, fmt.Sprintf("D%d", rowNum), row.Amount)
		f.SetCellValue(expenseSheet, fmt.Sprintf("E%d", rowNum), row.Note)
		f.SetCellValue(expenseSheet, fmt.Sprintf("F%d", rowNum), row.Trip)

		f.SetCellStyle(expenseSheet, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("C%d", rowNum), dataStyle)
		f.SetCellStyle(expenseSheet, fmt.Sprintf("D%d", rowNum), fmt.Sprintf("D%d", rowNum), amountStyle)
		f.SetCellStyle(expenseSheet, fmt.Sprintf("E%d", rowNum), fmt.Sprintf("F%d", rowNum), dataStyle)

		totalAmount += row.Amount
	}

	// Total row
	totalRowNum := len(expenseRows) + 4
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E5E7EB"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})
	f.MergeCell(expenseSheet, fmt.Sprintf("A%d", totalRowNum), fmt.Sprintf("C%d", totalRowNum))
	f.SetCellValue(expenseSheet, fmt.Sprintf("A%d", totalRowNum), "TOTAL")
	f.SetCellValue(expenseSheet, fmt.Sprintf("D%d", totalRowNum), totalAmount)
	f.SetCellStyle(expenseSheet, fmt.Sprintf("A%d", totalRowNum), fmt.Sprintf("F%d", totalRowNum), totalStyle)

	// Set column widths
	f.SetColWidth(expenseSheet, "A", "A", 15)
	f.SetColWidth(expenseSheet, "B", "B", 25)
	f.SetColWidth(expenseSheet, "C", "C", 15)
	f.SetColWidth(expenseSheet, "D", "D", 15)
	f.SetColWidth(expenseSheet, "E", "E", 30)
	f.SetColWidth(expenseSheet, "F", "F", 15)

	// ===== CATEGORY SUMMARY SHEET =====
	categorySheet := "Category Summary"
	f.NewSheet(categorySheet)

	// Title
	f.MergeCell(categorySheet, "A1", "D1")
	f.SetCellValue(categorySheet, "A1", fmt.Sprintf("Category Summary - %s", monthName))
	f.SetCellStyle(categorySheet, "A1", "D1", titleStyle)
	f.SetRowHeight(categorySheet, 1, 30)

	// Headers
	catHeaders := []string{"Category", "Count", "Total (₹)", "Percentage"}
	for i, header := range catHeaders {
		cell := fmt.Sprintf("%c3", 'A'+i)
		f.SetCellValue(categorySheet, cell, header)
		f.SetCellStyle(categorySheet, cell, cell, headerStyle)
	}

	// Category data
	for i, summary := range categorySummaries {
		rowNum := i + 4
		percentage := 0.0
		if totalAmount > 0 {
			percentage = (summary.Total / totalAmount) * 100
		}

		f.SetCellValue(categorySheet, fmt.Sprintf("A%d", rowNum), summary.Category)
		f.SetCellValue(categorySheet, fmt.Sprintf("B%d", rowNum), summary.Count)
		f.SetCellValue(categorySheet, fmt.Sprintf("C%d", rowNum), summary.Total)
		f.SetCellValue(categorySheet, fmt.Sprintf("D%d", rowNum), fmt.Sprintf("%.1f%%", percentage))

		f.SetCellStyle(categorySheet, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("B%d", rowNum), dataStyle)
		f.SetCellStyle(categorySheet, fmt.Sprintf("C%d", rowNum), fmt.Sprintf("C%d", rowNum), amountStyle)
		f.SetCellStyle(categorySheet, fmt.Sprintf("D%d", rowNum), fmt.Sprintf("D%d", rowNum), dataStyle)
	}

	// Total row for category
	catTotalRowNum := len(categorySummaries) + 4
	f.SetCellValue(categorySheet, fmt.Sprintf("A%d", catTotalRowNum), "TOTAL")
	f.SetCellValue(categorySheet, fmt.Sprintf("B%d", catTotalRowNum), len(expenses))
	f.SetCellValue(categorySheet, fmt.Sprintf("C%d", catTotalRowNum), totalAmount)
	f.SetCellValue(categorySheet, fmt.Sprintf("D%d", catTotalRowNum), "100%")
	f.SetCellStyle(categorySheet, fmt.Sprintf("A%d", catTotalRowNum), fmt.Sprintf("D%d", catTotalRowNum), totalStyle)

	// Set column widths
	f.SetColWidth(categorySheet, "A", "A", 18)
	f.SetColWidth(categorySheet, "B", "B", 12)
	f.SetColWidth(categorySheet, "C", "C", 15)
	f.SetColWidth(categorySheet, "D", "D", 12)

	// ===== PIE CHART =====
	if len(categorySummaries) > 0 {
		// Add pie chart
		chartDataStart := 4
		chartDataEnd := chartDataStart + len(categorySummaries) - 1

		pieChart := &excelize.Chart{
			Type: excelize.Pie,
			Series: []excelize.ChartSeries{
				{
					Name:       fmt.Sprintf("'%s'!$C$3", categorySheet),
					Categories: fmt.Sprintf("'%s'!$A$%d:$A$%d", categorySheet, chartDataStart, chartDataEnd),
					Values:     fmt.Sprintf("'%s'!$C$%d:$C$%d", categorySheet, chartDataStart, chartDataEnd),
				},
			},
			Format: excelize.GraphicOptions{
				OffsetX: 15,
				OffsetY: 10,
			},
			Legend: excelize.ChartLegend{
				Position:      "right",
				ShowLegendKey: false,
			},
			Title: []excelize.RichTextRun{
				{
					Text: fmt.Sprintf("Expenses by Category - %s", monthName),
				},
			},
			PlotArea: excelize.ChartPlotArea{
				ShowCatName:     true,
				ShowLeaderLines: true,
				ShowPercent:     true,
				ShowVal:         false,
			},
		}

		f.AddChart(categorySheet, "F3", pieChart)
	}

	// Set active sheet
	f.SetActiveSheet(0)

	// Generate filename
	filename := fmt.Sprintf("Expense_Report_%s_%s.xlsx", month, year)

	// Set response headers
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Cache-Control", "no-cache")

	// Write to response
	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report"})
		return
	}
}
