package service

import (
	"context"
	"errors"
	"time"

	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/repository"
)

type CashflowService struct {
	repo *repository.CashflowRepository
}

func NewCashflowService(repo *repository.CashflowRepository) *CashflowService {
	return &CashflowService{repo: repo}
}

// CreateManualCashflow validates and inserts a manual cashflow entry
func (s *CashflowService) CreateManualCashflow(ctx context.Context, req domain.CashflowRequest, creator string) (*domain.Cashflow, error) {
	if req.CashflowType != "INC" && req.CashflowType != "EXP" {
		return nil, errors.New("cashflow type must be either 'INC' or 'EXP'")
	}
	if req.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if req.Category == "" {
		return nil, errors.New("category is required")
	}

	// Parse flow date
	flowDate := time.Now()
	if req.FlowDate != "" {
		parsedDate, err := time.Parse("2006-01-02", req.FlowDate)
		if err != nil {
			return nil, errors.New("invalid flow date format, must be YYYY-MM-DD")
		}
		// Convert to local start of day
		flowDate = time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 12, 0, 0, 0, time.Local)
	}

	cashflow := &domain.Cashflow{
		CashflowType: req.CashflowType,
		Amount:       req.Amount,
		Category:     req.Category,
		FlowDate:     flowDate,
		CreatedBy:    &creator,
		UpdatedBy:    &creator,
	}

	if req.Description != "" {
		desc := req.Description
		cashflow.Description = &desc
	}

	err := s.repo.Insert(ctx, cashflow)
	if err != nil {
		return nil, err
	}

	return cashflow, nil
}

// GetAllCashflows retrieves list of cashflows with filters
func (s *CashflowService) GetAllCashflows(ctx context.Context, typeFilter string, categoryFilter string, startDate string, endDate string) ([]*domain.Cashflow, error) {
	return s.repo.FindAll(ctx, typeFilter, categoryFilter, startDate, endDate)
}

// DeleteCashflow soft deletes a manual cashflow entry
func (s *CashflowService) DeleteCashflow(ctx context.Context, id int, updatedBy string) error {
	return s.repo.SoftDelete(ctx, id, updatedBy)
}

// GetFinanceSummary retrieves aggregates for finance indicators
func (s *CashflowService) GetFinanceSummary(ctx context.Context) (*domain.FinanceSummary, error) {
	return s.repo.GetSummary(ctx)
}

// GetChartData aggregates cashflows for visualization
func (s *CashflowService) GetChartData(ctx context.Context, period string) ([]*domain.FinanceChartItem, error) {
	if period != "monthly" {
		period = "daily"
	}
	return s.repo.GetChartData(ctx, period)
}
