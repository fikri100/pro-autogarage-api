package domain

import "time"

// Customer represents the database entity
type Customer struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Phone         string    `json:"phone"`
	Address       *string   `json:"address"` // nullable
	Email         *string   `json:"email"`   // nullable
	IsSelfService bool      `json:"isSelfService"`
	Password      *string   `json:"-"` // never expose password in JSON
	Status        string    `json:"status"`
	CreatedBy     *string   `json:"createdBy"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedBy     *string   `json:"updatedBy"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// CustomerRequest represents the payload from the client
type CustomerRequest struct {
	Name          string  `json:"name"`
	Phone         string  `json:"phone"`
	Address       *string `json:"address"`
	Email         *string `json:"email"`
	IsSelfService bool    `json:"isSelfService"`
	Password      *string `json:"password"`
}
