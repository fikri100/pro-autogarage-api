package domain

import "time"

// Cashflow represents the cashflows database table
type Cashflow struct {
	ID            int        `json:"id"`
	CashflowType  string     `json:"cashflowType"` // 'INC' or 'EXP'
	Amount        float64    `json:"amount"`
	Category      string     `json:"category"` // 'SERVICE', 'ELECTRICITY', 'SALARY', 'STOCK', 'OTHER'
	Description   *string    `json:"description"`
	TransactionID *int       `json:"transactionId"` // Nullable
	FlowDate      time.Time  `json:"flowDate"`
	Status        string     `json:"status"`
	CreatedBy     *string    `json:"createdBy"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedBy     *string    `json:"updatedBy"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// CashflowRequest represents the payload to create a cashflow entry manually
type CashflowRequest struct {
	CashflowType string  `json:"cashflowType"` // 'INC' or 'EXP'
	Amount       float64 `json:"amount"`
	Category     string  `json:"category"`
	Description  string  `json:"description"`
	FlowDate     string  `json:"flowDate"` // Format YYYY-MM-DD
}

// FinanceSummary represents the high-level metrics for finance dashboard KPI cards
type FinanceSummary struct {
	TotalIncome          float64 `json:"totalIncome"`
	TotalExpense         float64 `json:"totalExpense"`
	NetCashflow          float64 `json:"netCashflow"`
	GrossProfit          float64 `json:"grossProfit"`
	TotalServiceRevenue  float64 `json:"totalServiceRevenue"`
	TotalSparepartSales  float64 `json:"totalSparepartSales"`
	TotalSparepartCOGS   float64 `json:"totalSparepartCOGS"`
}

// FinanceChartItem represents an aggregated daily or monthly data point for line/bar charts
type FinanceChartItem struct {
	Label   string  `json:"label"` // e.g., '2026-05-25' or 'May 2026'
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
}
