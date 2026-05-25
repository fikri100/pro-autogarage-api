package service

import (
	"context"

	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/repository"
)

type DashboardService struct {
	repo *repository.DashboardRepository
}

func NewDashboardService(repo *repository.DashboardRepository) *DashboardService {
	return &DashboardService{repo: repo}
}

// GetDashboardSummary consolidates KPI stats, recent pending bookings, and live active work orders
func (s *DashboardService) GetDashboardSummary(ctx context.Context) (*domain.DashboardSummary, error) {
	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	bookings, err := s.repo.GetRecentBookings(ctx, 10) // fetch top 10 pending bookings
	if err != nil {
		return nil, err
	}

	workOrders, err := s.repo.GetActiveWorkOrders(ctx)
	if err != nil {
		return nil, err
	}

	summary := &domain.DashboardSummary{
		Stats:           stats,
		RecentBookings:  bookings,
		ActiveWorkOrders: workOrders,
	}

	return summary, nil
}
