package domain

import "time"

// User represents the full user data with JOINed tables
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	RoleID       int       `json:"roleId"`
	RoleName     string    `json:"roleName"`
	EmployeeID   int       `json:"employeeId"`
	EmployeeName string    `json:"employeeName"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
}

// UserRequest payload for creating/updating a user
type UserRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	RoleID       int    `json:"roleId"`
	EmployeeID   int    `json:"employeeId"`
	EmployeeName string `json:"employeeName,omitempty"`
	Position     string `json:"position,omitempty"`
}

// Role represents a system role
type Role struct {
	ID          int    `json:"id"`
	RoleName    string `json:"roleName"`
	Permissions string `json:"permissions"` // Stored as JSONB string in DB
}


// Employee represents a staff member
type Employee struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Address    string `json:"address"`
	PositionID int    `json:"positionId"`
	Position   string `json:"position"`
}

// EmployeeRequest represents request payload to create/update an employee
type EmployeeRequest struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Address    string `json:"address"`
	PositionID int    `json:"positionId"`
	Position   string `json:"position"`
}


// MenuItem represents a dynamic role-based menu item
type MenuItem struct {
	ID         int        `json:"id"`
	Label      string     `json:"label"`
	Icon       string     `json:"icon"`
	RouterLink string     `json:"routerLink"`
	ParentID   *int       `json:"parentId,omitempty"`
	Children   []MenuItem `json:"children"`
	IsOpen     bool       `json:"isOpen"`
}

// FlatMenu represents a flat menu item retrieved from database before restructuring
type FlatMenu struct {
	ID         int
	Label      string
	Icon       string
	RouterLink string
	ParentID   *int
}

// MenuResponse represents a simple menu item format for the list endpoint
type MenuResponse struct {
	ID         int    `json:"id"`
	Label      string `json:"label"`
	Icon       string `json:"icon"`
	RouterLink string `json:"routerLink"`
	ParentID   *int   `json:"parentId,omitempty"`
}

// RoleMenuMappingRequest represents menu mapping request payload
type RoleMenuMappingRequest struct {
	MenuIDs []int `json:"menuIds"`
}

// LoginRequest payload for administrator login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserResponse represents full user details including mapped menus after login
type UserResponse struct {
	ID           int        `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	RoleID       int        `json:"roleId"`
	RoleName     string     `json:"roleName"`
	EmployeeID   int        `json:"employeeId"`
	EmployeeName string     `json:"employeeName"`
	Status       string     `json:"status"`
	Menus        []MenuItem `json:"menus"`
	Token        string     `json:"token,omitempty"`
}

// RoleRequest represents role payload for CRUD operations
type RoleRequest struct {
	RoleName string `json:"roleName"`
}

