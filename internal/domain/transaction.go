package domain

import "time"

type Transaction struct {
	ID              int                  `json:"id"`
	WorkOrderID     int                  `json:"workOrderId"`
	InvoiceNumber   string               `json:"invoiceNumber"`
	TotalAmount     float64              `json:"totalAmount"`
	Discount        float64              `json:"discount"` // we can add discount field
	Tax             float64              `json:"tax"`      // we can add tax field (PPN 11%)
	PaymentMethodID *int                 `json:"paymentMethodId"`
	PaymentMethod   *string              `json:"paymentMethod"`
	PaymentStatusID int                  `json:"paymentStatusId"`
	PaymentStatus   string               `json:"paymentStatus"` // UNPAID, PAID
	TransactionDate time.Time            `json:"transactionDate"`
	Status          string               `json:"status"`
	CreatedBy       *string              `json:"createdBy"`
	CreatedAt       time.Time            `json:"createdAt"`
	UpdatedBy       *string              `json:"updatedBy"`
	UpdatedAt       time.Time            `json:"updatedAt"`
	Details         []*TransactionDetail `json:"details,omitempty"`
}

type TransactionDetail struct {
	ID                 int       `json:"id"`
	TransactionID      int       `json:"transactionId"`
	ProductID          int       `json:"productId"`
	ProductCode        string    `json:"productCode"`
	ProductName        string    `json:"productName"`
	ProductType        string    `json:"productType"` // 'SPR' or 'SRV'
	ProductCategory    string    `json:"productCategory"`
	PurchasePrice      float64   `json:"purchasePrice"` // useful for financial calculations
	Quantity           int       `json:"quantity"`
	PriceAtTransaction float64   `json:"priceAtTransaction"`
	Subtotal           float64   `json:"subtotal"`
	Status             string    `json:"status"`
	CreatedBy          *string   `json:"createdBy"`
	CreatedAt          time.Time `json:"createdAt"`
}

type TransactionDetailRequest struct {
	ProductID          int     `json:"productId"`
	Quantity           int     `json:"quantity"`
	PriceAtTransaction float64 `json:"priceAtTransaction"`
}

type PaymentRequest struct {
	PaymentMethodID int                         `json:"paymentMethodId"`
	PaymentMethod   string                      `json:"paymentMethod"`
	Discount        float64                     `json:"discount"`
	Details         []*TransactionDetailRequest `json:"details"`
}
