package service

import (
	"context"
	"errors"
	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/repository"
)

type TransactionService struct {
	transactionRepo *repository.TransactionRepository
}

func NewTransactionService(transactionRepo *repository.TransactionRepository) *TransactionService {
	return &TransactionService{transactionRepo: transactionRepo}
}

// GetReadyWorkOrders lists completed WOs ready to be billed with pagination and search
func (s *TransactionService) GetReadyWorkOrders(ctx context.Context, search string, page int, limit int) ([]*domain.WorkOrder, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	return s.transactionRepo.FindAllReadyForCashier(ctx, search, limit, offset)
}

// GetTransactionByWO loads draft invoice details
func (s *TransactionService) GetTransactionByWO(ctx context.Context, woID int) (*domain.Transaction, []*domain.TransactionDetail, error) {
	return s.transactionRepo.GetTransactionByWO(ctx, woID)
}

// FinalizePayment processes the billing, calculates tax, and commits DB transaction
func (s *TransactionService) FinalizePayment(ctx context.Context, transID int, req domain.PaymentRequest, adminUser string) error {
	if req.PaymentMethod != "Tunai" && req.PaymentMethod != "Transfer Bank" && req.PaymentMethod != "QRIS" {
		return errors.New("invalid payment method. Choose 'Tunai', 'Transfer Bank', or 'QRIS'")
	}

	if len(req.Details) == 0 {
		return errors.New("invoice details cannot be empty")
	}

	// Recalculate everything on the server to prevent tamper
	var subtotal float64
	var details []*domain.TransactionDetail

	for _, reqDetail := range req.Details {
		if reqDetail.Quantity <= 0 {
			return errors.New("quantity must be greater than 0")
		}

		itemSubtotal := float64(reqDetail.Quantity) * reqDetail.PriceAtTransaction
		subtotal += itemSubtotal

		details = append(details, &domain.TransactionDetail{
			ProductID:          reqDetail.ProductID,
			Quantity:           reqDetail.Quantity,
			PriceAtTransaction: reqDetail.PriceAtTransaction,
			Subtotal:           itemSubtotal,
		})
	}

	discount := req.Discount
	if discount < 0 {
		discount = 0
	}

	netAmount := subtotal - discount
	if netAmount < 0 {
		netAmount = 0
	}

	// Calculate 11% PPN tax reactively
	tax := netAmount * 0.11
	grandTotal := netAmount + tax

	return s.transactionRepo.FinalizePaymentTx(ctx, transID, req.PaymentMethod, discount, tax, grandTotal, details, adminUser)
}
