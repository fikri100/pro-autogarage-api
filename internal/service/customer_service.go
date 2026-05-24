package service

import (
	"context"
	"errors"
	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/repository"
)

type CustomerService struct {
	repo *repository.CustomerRepository
}

func NewCustomerService(repo *repository.CustomerRepository) *CustomerService {
	return &CustomerService{repo: repo}
}

// CreateCustomer validates and delegates creation to repository
func (s *CustomerService) CreateCustomer(ctx context.Context, req domain.CustomerRequest, adminUser string) (*domain.Customer, error) {
	if req.Name == "" || req.Phone == "" {
		return nil, errors.New("name and phone are required")
	}

	customer := &domain.Customer{
		Name:          req.Name,
		Phone:         req.Phone,
		Address:       req.Address,
		Email:         req.Email,
		IsSelfService: req.IsSelfService,
		Password:      req.Password,
		CreatedBy:     &adminUser,
		UpdatedBy:     &adminUser,
	}

	err := s.repo.Insert(ctx, customer)
	if err != nil {
		return nil, err
	}
	return customer, nil
}

// GetAllCustomers returns all active customers
func (s *CustomerService) GetAllCustomers(ctx context.Context) ([]*domain.Customer, error) {
	return s.repo.FindAll(ctx)
}

// GetCustomerByID returns a customer by ID
func (s *CustomerService) GetCustomerByID(ctx context.Context, id int) (*domain.Customer, error) {
	return s.repo.FindByID(ctx, id)
}

// UpdateCustomer validates and updates existing customer
func (s *CustomerService) UpdateCustomer(ctx context.Context, id int, req domain.CustomerRequest, adminUser string) error {
	if req.Name == "" || req.Phone == "" {
		return errors.New("name and phone are required")
	}

	customer := &domain.Customer{
		ID:            id,
		Name:          req.Name,
		Phone:         req.Phone,
		Address:       req.Address,
		Email:         req.Email,
		IsSelfService: req.IsSelfService,
		UpdatedBy:     &adminUser,
	}

	return s.repo.Update(ctx, customer)
}

// DeleteCustomer soft-deletes a customer
func (s *CustomerService) DeleteCustomer(ctx context.Context, id int, adminUser string) error {
	return s.repo.SoftDelete(ctx, id, adminUser)
}
