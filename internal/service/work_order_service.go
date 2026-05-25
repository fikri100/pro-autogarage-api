package service

import (
	"context"
	"errors"
	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/repository"
)

type WorkOrderService struct {
	woRepo          *repository.WorkOrderRepository
	transactionRepo *repository.TransactionRepository
}

func NewWorkOrderService(woRepo *repository.WorkOrderRepository, transactionRepo *repository.TransactionRepository) *WorkOrderService {
	return &WorkOrderService{
		woRepo:          woRepo,
		transactionRepo: transactionRepo,
	}
}

// CreateWorkOrder creates a direct walk-in work order
func (s *WorkOrderService) CreateWorkOrder(ctx context.Context, req domain.WorkOrderRequest, createdBy string) (*domain.WorkOrder, error) {
	if req.VehicleID <= 0 {
		return nil, errors.New("vehicleId is required")
	}

	wo := &domain.WorkOrder{
		BookingID:  req.BookingID,
		VehicleID:  req.VehicleID,
		MechanicID: req.MechanicID,
		WorkStatus: "IN_PROGRESS",
		Notes:      req.Notes,
		CreatedBy:  &createdBy,
		UpdatedBy:  &createdBy,
	}

	err := s.woRepo.Insert(ctx, wo)
	if err != nil {
		return nil, err
	}

	return s.woRepo.FindByID(ctx, wo.ID)
}

// GetAllActiveWorkOrders retrieves active ones for the dashboard
func (s *WorkOrderService) GetAllActiveWorkOrders(ctx context.Context) ([]*domain.WorkOrder, error) {
	return s.woRepo.FindAllActive(ctx)
}

// GetWorkOrder returns a specific WO
func (s *WorkOrderService) GetWorkOrder(ctx context.Context, id int) (*domain.WorkOrder, error) {
	return s.woRepo.FindByID(ctx, id)
}

// AssignMechanic updates mechanic details and notes
func (s *WorkOrderService) AssignMechanic(ctx context.Context, id int, mechanicID *int, notes *string, updatedBy string) error {
	wo, err := s.woRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if wo.WorkStatus != "IN_PROGRESS" {
		return errors.New("cannot assign mechanics to a finished or paid work order")
	}

	return s.woRepo.AssignMechanic(ctx, id, mechanicID, notes, "IN_PROGRESS", updatedBy)
}

// SaveEstimation saves estimated products/services under a draft invoice (transaction)
func (s *WorkOrderService) SaveEstimation(ctx context.Context, id int, details []*domain.TransactionDetail, createdBy string) error {
	wo, err := s.woRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if wo.WorkStatus != "IN_PROGRESS" {
		return errors.New("cannot edit estimations on a completed or paid work order")
	}

	return s.transactionRepo.CreateUnpaidTransaction(ctx, id, details, createdBy)
}

// CompleteWorkOrder finishes mechanic tasks and moves status to COMPLETED
func (s *WorkOrderService) CompleteWorkOrder(ctx context.Context, id int, updatedBy string) error {
	wo, err := s.woRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if wo.WorkStatus != "IN_PROGRESS" {
		return errors.New("only IN_PROGRESS work orders can be completed")
	}

	return s.woRepo.UpdateStatus(ctx, id, "COMPLETED", updatedBy)
}
