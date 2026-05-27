package repository

import (
	"context"
	"database/sql"
	"errors"
	"pro-autogarage-api/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindAllUsers gets users with their role and employee details
func (r *UserRepository) FindAllUsers(ctx context.Context) ([]*domain.User, error) {
	query := `
		SELECT u.id, u.username, u.role_id, r.role_name, u.employee_id, e.name as employee_name, u.status, u.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		JOIN employees e ON u.employee_id = e.id
		WHERE u.status = 'Y'
		ORDER BY u.id ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.RoleID, &u.RoleName, &u.EmployeeID, &u.EmployeeName, &u.Status, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

// InsertUser creates a new user
func (r *UserRepository) InsertUser(ctx context.Context, u *domain.UserRequest, createdBy string) error {
	query := `
		INSERT INTO users (username, password, role_id, employee_id, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query, u.Username, u.Password, u.RoleID, u.EmployeeID, createdBy, createdBy)
	return err
}

// InsertEmployee creates a new employee and returns the generated ID
func (r *UserRepository) InsertEmployee(ctx context.Context, name, position, phone, createdBy string) (int, error) {
	query := `
		INSERT INTO employees (name, position, phone, status, created_by, updated_by)
		VALUES ($1, $2, $3, 'Y', $4, $5)
		RETURNING id
	`
	var id int
	err := r.db.QueryRowContext(ctx, query, name, position, phone, createdBy, createdBy).Scan(&id)
	return id, err
}

// UpdateUser updates user details (except password)
func (r *UserRepository) UpdateUser(ctx context.Context, id int, u *domain.UserRequest, updatedBy string) error {
	query := `
		UPDATE users
		SET role_id = $1, employee_id = $2, updated_by = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4 AND status = 'Y'
	`
	res, err := r.db.ExecContext(ctx, query, u.RoleID, u.EmployeeID, updatedBy, id)
	if err != nil {
		return err
	}
	
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("user not found")
	}
	return nil
}

// GetRoles gets all active roles
func (r *UserRepository) GetRoles(ctx context.Context) ([]*domain.Role, error) {
	query := `SELECT id, role_name, COALESCE(permissions::text, '{}') as permissions FROM roles WHERE status = 'Y' ORDER BY id ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*domain.Role
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.RoleName, &role.Permissions); err != nil {
			return nil, err
		}
		roles = append(roles, &role)
	}
	return roles, nil
}


// GetEmployees gets all active employees
func (r *UserRepository) GetEmployees(ctx context.Context) ([]*domain.Employee, error) {
	query := `SELECT id, name, position FROM employees WHERE status = 'Y' ORDER BY name ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emps []*domain.Employee
	for rows.Next() {
		var e domain.Employee
		if err := rows.Scan(&e.ID, &e.Name, &e.Position); err != nil {
			return nil, err
		}
		emps = append(emps, &e)
	}
	return emps, nil
}
