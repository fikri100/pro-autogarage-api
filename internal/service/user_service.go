package service

import (
	"context"
	"errors"
	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// GetAllUsers retrieves all users
func (s *UserService) GetAllUsers(ctx context.Context) ([]*domain.User, error) {
	return s.repo.FindAllUsers(ctx)
}

// CreateUser validates and creates a new user, hashing the password
func (s *UserService) CreateUser(ctx context.Context, req domain.UserRequest, adminUser string) error {
	if req.Username == "" || req.Password == "" || req.RoleID == 0 || req.EmployeeID == 0 {
		return errors.New("username, password, roleId, and employeeId are required")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	req.Password = string(hashedPassword)

	return s.repo.InsertUser(ctx, &req, adminUser)
}

// UpdateUser validates and updates a user (no password update in this simple flow)
func (s *UserService) UpdateUser(ctx context.Context, id int, req domain.UserRequest, adminUser string) error {
	if req.RoleID == 0 || req.EmployeeID == 0 {
		return errors.New("roleId and employeeId are required")
	}
	return s.repo.UpdateUser(ctx, id, &req, adminUser)
}

// GetRoles gets all active roles
func (s *UserService) GetRoles(ctx context.Context) ([]*domain.Role, error) {
	return s.repo.GetRoles(ctx)
}

// UpdateRolePermissions updates the JSON permissions matrix
func (s *UserService) UpdateRolePermissions(ctx context.Context, id int, permissions string, adminUser string) error {
	if permissions == "" {
		permissions = "{}"
	}
	return s.repo.UpdateRolePermissions(ctx, id, permissions, adminUser)
}

// GetEmployees gets all active employees
func (s *UserService) GetEmployees(ctx context.Context) ([]*domain.Employee, error) {
	return s.repo.GetEmployees(ctx)
}
