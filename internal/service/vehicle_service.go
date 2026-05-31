package service

import (
	"context"
	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/repository"
)

type VehicleService struct {
	repo *repository.VehicleRepository
}

func NewVehicleService(repo *repository.VehicleRepository) *VehicleService {
	return &VehicleService{repo: repo}
}

func (s *VehicleService) GetAllVehicles(ctx context.Context, customerID int) ([]domain.Vehicle, error) {
	return s.repo.FindAll(ctx, customerID)
}

func (s *VehicleService) CreateVehicle(ctx context.Context, req domain.VehicleRequest) (int, error) {
	return s.repo.Insert(ctx, &req)
}

func (s *VehicleService) UpdateVehicle(ctx context.Context, id int, req domain.UpdateVehicleRequest) error {
	return s.repo.Update(ctx, id, &req)
}

func (s *VehicleService) DeleteVehicle(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
