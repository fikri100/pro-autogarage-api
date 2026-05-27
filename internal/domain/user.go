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
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Position string `json:"position"`
}
