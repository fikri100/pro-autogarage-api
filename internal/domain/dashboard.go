package domain

// DashboardStats represents KPI metrics shown on the dashboard cards
type DashboardStats struct {
	TotalCustomers  int     `json:"totalCustomers"`
	ActiveWorkOrders int     `json:"activeWorkOrders"`
	TodayRevenue    float64 `json:"todayRevenue"`
	PendingBookings int     `json:"pendingBookings"`
}

// DashboardSummary represents the entire payload returned to render the executive dashboard
type DashboardSummary struct {
	Stats           DashboardStats `json:"stats"`
	RecentBookings  []*Booking     `json:"recentBookings"`
	ActiveWorkOrders []*WorkOrder  `json:"activeWorkOrders"`
}
