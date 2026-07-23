package domain

import "time"

// Product represents the products database table
type Product struct {
	ID            int       `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	ItemTypeID    int       `json:"itemTypeId"`
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
	ItemTypeID    int      `json:"itemTypeId"`
	ItemType      string   `json:"itemType"`
	Category      *string  `json:"category"`
	PurchasePrice *float64 `json:"purchasePrice"`
	SalePrice     float64  `json:"salePrice"`
	StockQuantity *int     `json:"stockQuantity"`
	MinStockLimit *int     `json:"minStockLimit"`
}

// RestockRequest represents the client payload for restocking a product
type RestockRequest struct {
	ProductID     int     `json:"productId"`
	Quantity      int     `json:"quantity"`
	PurchasePrice float64 `json:"purchasePrice"`
	ReferenceID   string  `json:"referenceId"`
	RecordExpense bool    `json:"recordExpense"`
}

// StockLog represents a single record in the stock_logs database table
type StockLog struct {
	ID          int       `json:"id"`
	ProductID   int       `json:"productId"`
	LogType     string    `json:"logType"` // 'IN' or 'OUT'
	Quantity    int       `json:"quantity"`
	ReferenceID string    `json:"referenceId"`
	Status      string    `json:"status"`
	CreatedBy   string    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedBy   string    `json:"updatedBy"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Category represents a product/service category
type Category struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	ItemTypeID   int    `json:"itemTypeId"`
	ItemTypeName string `json:"itemTypeName"`
}

// CategoryRequest represents request payload to create/update a category
type CategoryRequest struct {
	Name       string `json:"name"`
	ItemTypeID int    `json:"itemTypeId"`
}


