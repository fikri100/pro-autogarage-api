package service

import (
	"context"
	"database/sql"
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
	if req.Username == "" || req.Password == "" || req.RoleID == 0 {
		return errors.New("username, password, and roleId are required")
	}

	// If EmployeeID is not provided, create a new employee first
	if req.EmployeeID == 0 {
		if req.EmployeeName == "" || req.Position == "" {
			return errors.New("employeeName and position are required to create a new employee")
		}
		
		newEmpID, err := s.repo.InsertEmployee(ctx, req.EmployeeName, req.Position, "-", adminUser)
		if err != nil {
			return err
		}
		req.EmployeeID = newEmpID
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


// GetEmployees gets all active employees
func (s *UserService) GetEmployees(ctx context.Context) ([]*domain.Employee, error) {
	return s.repo.GetEmployees(ctx)
}

// Login validates username and password and returns UserResponse with menus
func (s *UserService) Login(ctx context.Context, req domain.LoginRequest) (*domain.UserResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, errors.New("Username dan password wajib diisi")
	}

	u, err := s.repo.FindUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("Username atau password salah")
		}
		return nil, err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password))
	if err != nil {
		if u.PasswordHash != req.Password {
			return nil, errors.New("Username atau password salah")
		}
	}

	// Fetch menus
	menus, err := s.repo.FindMenusForRole(ctx, u.RoleID)
	if err != nil {
		return nil, err
	}
	u.Menus = menus

	return u, nil
}

// GetAllMenus retrieves all active menus
func (s *UserService) GetAllMenus(ctx context.Context) ([]domain.MenuResponse, error) {
	return s.repo.FindAllMenus(ctx)
}

// GetRoleMenus retrieves active menu IDs mapped to a role
func (s *UserService) GetRoleMenus(ctx context.Context, roleID int) ([]int, error) {
	return s.repo.FindRoleMenuIDs(ctx, roleID)
}

// UpdateRoleMenus updates menu mappings for a role
func (s *UserService) UpdateRoleMenus(ctx context.Context, roleID int, req domain.RoleMenuMappingRequest) error {
	return s.repo.UpdateRoleMenus(ctx, roleID, req.MenuIDs)
}

// CreateRole creates a new role
func (s *UserService) CreateRole(ctx context.Context, req domain.RoleRequest) (int, error) {
	if req.RoleName == "" {
		return 0, errors.New("Nama role wajib diisi")
	}
	return s.repo.InsertRole(ctx, req.RoleName)
}

// UpdateRole updates an existing role
func (s *UserService) UpdateRole(ctx context.Context, id int, req domain.RoleRequest) error {
	if req.RoleName == "" {
		return errors.New("Nama role wajib diisi")
	}
	return s.repo.UpdateRole(ctx, id, req.RoleName)
}

// DeleteRole soft deletes a role
func (s *UserService) DeleteRole(ctx context.Context, id int) error {
	return s.repo.DeleteRole(ctx, id)
}

// CreateEmployee creates a new employee record
func (s *UserService) CreateEmployee(ctx context.Context, req domain.EmployeeRequest, adminUser string) (int, error) {
	if req.Name == "" || req.Phone == "" || req.Position == "" {
		return 0, errors.New("Nama, Nomor Telepon, dan Jabatan wajib diisi")
	}
	return s.repo.CreateEmployee(ctx, &req, adminUser)
}

// UpdateEmployee updates an existing employee record
func (s *UserService) UpdateEmployee(ctx context.Context, id int, req domain.EmployeeRequest, adminUser string) error {
	if req.Name == "" || req.Phone == "" || req.Position == "" {
		return errors.New("Nama, Nomor Telepon, dan Jabatan wajib diisi")
	}
	return s.repo.UpdateEmployee(ctx, id, &req, adminUser)
}

// DeleteEmployee soft deletes an employee
func (s *UserService) DeleteEmployee(ctx context.Context, id int, adminUser string) error {
	return s.repo.DeleteEmployee(ctx, id, adminUser)
}


