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

// FindAllUsers gets users with their role and employee details with pagination and search
func (r *UserRepository) FindAllUsers(ctx context.Context, search string, limit, offset int) ([]*domain.User, int, error) {
	searchParam := "%" + search + "%"

	countQuery := `
		SELECT COUNT(*)
		FROM users u
		JOIN roles r ON u.role_id = r.id
		JOIN employees e ON u.employee_id = e.id
		WHERE u.status = 'Y' AND (u.username ILIKE $1 OR e.name ILIKE $1 OR r.role_name ILIKE $1)
	`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, searchParam).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT u.id, u.username, u.role_id, r.role_name, u.employee_id, e.name as employee_name, u.status, u.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		JOIN employees e ON u.employee_id = e.id
		WHERE u.status = 'Y' AND (u.username ILIKE $1 OR e.name ILIKE $1 OR r.role_name ILIKE $1)
		ORDER BY u.id ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, searchParam, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.RoleID, &u.RoleName, &u.EmployeeID, &u.EmployeeName, &u.Status, &u.CreatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, &u)
	}
	return users, total, nil
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


// GetEmployees gets all active employees with pagination and search
func (r *UserRepository) GetEmployees(ctx context.Context, search string, limit, offset int) ([]*domain.Employee, int, error) {
	// First, get the total count for pagination
	countQuery := `
		SELECT COUNT(*) 
		FROM employees 
		WHERE status = 'Y' AND (name ILIKE $1 OR phone ILIKE $1 OR position ILIKE $1)
	`
	var total int
	searchParam := "%" + search + "%"
	err := r.db.QueryRowContext(ctx, countQuery, searchParam).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Then, get the paginated data
	query := `
		SELECT id, name, phone, COALESCE(address, ''), position 
		FROM employees 
		WHERE status = 'Y' AND (name ILIKE $1 OR phone ILIKE $1 OR position ILIKE $1)
		ORDER BY name ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, searchParam, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var emps []*domain.Employee
	for rows.Next() {
		var e domain.Employee
		if err := rows.Scan(&e.ID, &e.Name, &e.Phone, &e.Address, &e.Position); err != nil {
			return nil, 0, err
		}
		emps = append(emps, &e)
	}
	return emps, total, nil
}

// CreateEmployee creates a new employee and returns the generated ID
func (r *UserRepository) CreateEmployee(ctx context.Context, e *domain.EmployeeRequest, createdBy string) (int, error) {
	query := `
		INSERT INTO employees (name, phone, address, position, status, created_by, updated_by)
		VALUES ($1, $2, $3, $4, 'Y', $5, $5)
		RETURNING id
	`
	var id int
	err := r.db.QueryRowContext(ctx, query, e.Name, e.Phone, e.Address, e.Position, createdBy).Scan(&id)
	return id, err
}

// UpdateEmployee updates an existing employee's details
func (r *UserRepository) UpdateEmployee(ctx context.Context, id int, e *domain.EmployeeRequest, updatedBy string) error {
	query := `
		UPDATE employees 
		SET name = $1, phone = $2, address = $3, position = $4, updated_by = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $6 AND status = 'Y'
	`
	_, err := r.db.ExecContext(ctx, query, e.Name, e.Phone, e.Address, e.Position, updatedBy, id)
	return err
}

// DeleteEmployee soft deletes an employee by setting status to 'N'
func (r *UserRepository) DeleteEmployee(ctx context.Context, id int, updatedBy string) error {
	query := `
		UPDATE employees 
		SET status = 'N', updated_by = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND status = 'Y'
	`
	_, err := r.db.ExecContext(ctx, query, updatedBy, id)
	return err
}


// FindUserByUsername retrieves a user record by username
func (r *UserRepository) FindUserByUsername(ctx context.Context, username string) (*domain.UserResponse, error) {
	query := `
		SELECT u.id, u.username, u.password, u.role_id, r.role_name, u.employee_id, e.name as employee_name, u.status
		FROM users u
		JOIN roles r ON u.role_id = r.id
		JOIN employees e ON u.employee_id = e.id
		WHERE LOWER(u.username) = LOWER($1) AND u.status = 'Y'
	`
	var u domain.UserResponse
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.RoleID, &u.RoleName, &u.EmployeeID, &u.EmployeeName, &u.Status,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindMenusForRole retrieves tree structures of menus mapped to a role
func (r *UserRepository) FindMenusForRole(ctx context.Context, roleID int) ([]domain.MenuItem, error) {
	query := `
		SELECT m.id, m.label, COALESCE(m.icon, ''), COALESCE(m.router_link, ''), m.parent_id
		FROM menus m
		JOIN role_menus rm ON m.id = rm.menu_id
		WHERE rm.role_id = $1 AND m.status = 'Y'
		ORDER BY m.parent_id ASC NULLS FIRST, m.sort_order ASC, m.id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flatMenus []domain.FlatMenu
	for rows.Next() {
		var fm domain.FlatMenu
		var parentID sql.NullInt64
		if err := rows.Scan(&fm.ID, &fm.Label, &fm.Icon, &fm.RouterLink, &parentID); err != nil {
			return nil, err
		}
		if parentID.Valid {
			val := int(parentID.Int64)
			fm.ParentID = &val
		}
		flatMenus = append(flatMenus, fm)
	}

	menuMap := make(map[int]*domain.MenuItem)
	var rootMenus []domain.MenuItem

	for _, fm := range flatMenus {
		menuMap[fm.ID] = &domain.MenuItem{
			ID:         fm.ID,
			Label:      fm.Label,
			Icon:       fm.Icon,
			RouterLink: fm.RouterLink,
			ParentID:   fm.ParentID,
			Children:   []domain.MenuItem{},
			IsOpen:     false,
		}
	}

	for _, fm := range flatMenus {
		item := menuMap[fm.ID]
		if fm.ParentID == nil {
			rootMenus = append(rootMenus, *item)
		} else {
			parentItem, exists := menuMap[*fm.ParentID]
			if exists {
				parentItem.Children = append(parentItem.Children, *item)
			}
		}
	}

	var finalRootMenus []domain.MenuItem
	for _, rm := range rootMenus {
		finalRootMenus = append(finalRootMenus, *menuMap[rm.ID])
	}

	if finalRootMenus == nil {
		finalRootMenus = []domain.MenuItem{}
	}

	return finalRootMenus, nil
}

// FindAllMenus retrieves all active system menus
func (r *UserRepository) FindAllMenus(ctx context.Context) ([]domain.MenuResponse, error) {
	query := "SELECT id, label, COALESCE(icon, ''), COALESCE(router_link, ''), parent_id FROM menus WHERE status = 'Y' ORDER BY parent_id ASC NULLS FIRST, sort_order ASC, id ASC"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var menus []domain.MenuResponse
	for rows.Next() {
		var m domain.MenuResponse
		var parentID sql.NullInt64
		if err := rows.Scan(&m.ID, &m.Label, &m.Icon, &m.RouterLink, &parentID); err != nil {
			return nil, err
		}
		if parentID.Valid {
			val := int(parentID.Int64)
			m.ParentID = &val
		}
		menus = append(menus, m)
	}
	if menus == nil {
		menus = []domain.MenuResponse{}
	}
	return menus, nil
}

// FindRoleMenuIDs retrieves raw menu IDs mapped to a specific role
func (r *UserRepository) FindRoleMenuIDs(ctx context.Context, roleID int) ([]int, error) {
	query := "SELECT menu_id FROM role_menus WHERE role_id = $1"
	rows, err := r.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var menuIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		menuIDs = append(menuIDs, id)
	}
	if menuIDs == nil {
		menuIDs = []int{}
	}
	return menuIDs, nil
}

// UpdateRoleMenus atomic transaction to update role and menu mapping
func (r *UserRepository) UpdateRoleMenus(ctx context.Context, roleID int, menuIDs []int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "DELETE FROM role_menus WHERE role_id = $1", roleID)
	if err != nil {
		return err
	}

	for _, mID := range menuIDs {
		_, err = tx.ExecContext(ctx, "INSERT INTO role_menus (role_id, menu_id) VALUES ($1, $2)", roleID, mID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// InsertRole creates a new role with default permissions template
func (r *UserRepository) InsertRole(ctx context.Context, roleName string) (int, error) {
	defaultPermissions := `{"dashboard":{"create":"N","read":"Y","update":"N","delete":"N"},"master":{"create":"N","read":"Y","update":"N","delete":"N"},"inventory":{"create":"N","read":"Y","update":"N","delete":"N"},"cashier":{"create":"N","read":"Y","update":"N","delete":"N"},"reports":{"create":"N","read":"Y","update":"N","delete":"N"}}`
	query := `
		INSERT INTO roles (role_name, permissions, created_by, updated_by)
		VALUES ($1, $2::jsonb, 'admin', 'admin')
		RETURNING id
	`
	var id int
	err := r.db.QueryRowContext(ctx, query, roleName, defaultPermissions).Scan(&id)
	return id, err
}

// UpdateRole updates an existing role's name
func (r *UserRepository) UpdateRole(ctx context.Context, id int, roleName string) error {
	query := `
		UPDATE roles 
		SET role_name = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND status = 'Y'
	`
	_, err := r.db.ExecContext(ctx, query, roleName, id)
	return err
}

// DeleteRole soft deletes a role by setting status to 'N'
func (r *UserRepository) DeleteRole(ctx context.Context, id int) error {
	query := `
		UPDATE roles 
		SET status = 'N', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// IsUsernameTaken checks if a username is already registered and active
func (r *UserRepository) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND status = 'Y')`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, username).Scan(&exists)
	return exists, err
}

// IsEmployeeMapped checks if an employee already has an active user account
func (r *UserRepository) IsEmployeeMapped(ctx context.Context, employeeID int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE employee_id = $1 AND status = 'Y')`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, employeeID).Scan(&exists)
	return exists, err
}


