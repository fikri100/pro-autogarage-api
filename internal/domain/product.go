package domain

import "time"

// Product represents the products database table
type Product struct {
	ID            int       `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	ItemType      string    `json:"itemType"` // 'SPR' (Sparepart) or 'SRV' (Service/Jasa)
	Category      *string   `json:"category"` // nullable
	PurchasePrice float64   `json:"purchasePrice"`
	SalePrice     float64   `json:"salePrice"`
	StockQuantity int       `json:"stockQuantity"`
	MinStockLimit int       `json:"minStockLimit"`
	Status        string    `json:"status"`
	CreatedBy     *string   `json:"createdBy"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedBy     *string   `json:"updatedBy"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ProductRequest represents the payload from clients to create/update a product
type ProductRequest struct {
	Code          string   `json:"code"`
	Name          string   `json:"name"`
	ItemType      string   `json:"itemType"`
	Category      *string  `json:"category"`
	PurchasePrice *float64 `json:"purchasePrice"`
	SalePrice     float64  `json:"salePrice"`
	StockQuantity *int     `json:"stockQuantity"`
	MinStockLimit *int     `json:"minStockLimit"`
}
