package service

import (
	"context"
	"errors"
	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/repository"
)

type ParamService struct {
	repo *repository.ParamRepository
}

func NewParamService(repo *repository.ParamRepository) *ParamService {
	return &ParamService{repo: repo}
}

// GetParamsByGroup validates and retrieves parameters for the specified group
func (s *ParamService) GetParamsByGroup(ctx context.Context, groupParam string) ([]*domain.Param, error) {
	if groupParam == "" {
		return nil, errors.New("group_param is required")
	}
	return s.repo.FindByGroup(ctx, groupParam)
}
