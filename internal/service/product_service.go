package service

import (
	"context"
	"errors"
	"pro-autogarage-api/internal/domain"
	"pro-autogarage-api/internal/repository"
)

type ProductService struct {
	repo *repository.ProductRepository
}

func NewProductService(repo *repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

// CreateProduct validates and delegates creation to repository
func (s *ProductService) CreateProduct(ctx context.Context, req domain.ProductRequest, adminUser string) (*domain.Product, error) {
	if req.Code == "" || req.Name == "" || req.ItemType == "" {
		return nil, errors.New("code, name, and itemType are required")
	}

	if req.ItemType != "SPR" && req.ItemType != "SRV" {
		return nil, errors.New("itemType must be either 'SPR' (Sparepart) or 'SRV' (Service/Jasa)")
	}

	// Check if product code already exists
	exists, err := s.repo.IsCodeExists(ctx, req.Code, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("product/service code already exists")
	}

	product := &domain.Product{
		Code:      req.Code,
		Name:      req.Name,
		ItemType:  req.ItemType,
		Category:  req.Category,
		SalePrice: req.SalePrice,
		CreatedBy: &adminUser,
		UpdatedBy: &adminUser,
	}

	// Apply item type specific logic
	if req.ItemType == "SRV" {
		product.PurchasePrice = 0
		product.StockQuantity = 0
		product.MinStockLimit = 0
	} else {
		if req.PurchasePrice != nil {
			product.PurchasePrice = *req.PurchasePrice
		}
		if req.StockQuantity != nil {
			product.StockQuantity = *req.StockQuantity
		}
		if req.MinStockLimit != nil {
			product.MinStockLimit = *req.MinStockLimit
		} else {
			product.MinStockLimit = 5 // default
		}
	}

	err = s.repo.Insert(ctx, product)
	if err != nil {
		return nil, err
	}
	return product, nil
}

// GetAllProducts retrieves products with optional filters
func (s *ProductService) GetAllProducts(ctx context.Context, search string, itemType string, lowStock bool, page int, limit int) ([]*domain.Product, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit
	return s.repo.FindAll(ctx, search, itemType, lowStock, limit, offset)
}

// GetProductByID returns a specific product by ID
func (s *ProductService) GetProductByID(ctx context.Context, id int) (*domain.Product, error) {
	return s.repo.FindByID(ctx, id)
}

// UpdateProduct validates and updates a product
func (s *ProductService) UpdateProduct(ctx context.Context, id int, req domain.ProductRequest, adminUser string) error {
	if req.Code == "" || req.Name == "" || req.ItemType == "" {
		return errors.New("code, name, and itemType are required")
	}

	if req.ItemType != "SPR" && req.ItemType != "SRV" {
		return errors.New("itemType must be either 'SPR' (Sparepart) or 'SRV' (Service/Jasa)")
	}

	// Check if product code already exists for other products
	exists, err := s.repo.IsCodeExists(ctx, req.Code, id)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("product/service code already exists")
	}

	product := &domain.Product{
		ID:        id,
		Code:      req.Code,
		Name:      req.Name,
		ItemType:  req.ItemType,
		Category:  req.Category,
		SalePrice: req.SalePrice,
		UpdatedBy: &adminUser,
	}

	// Apply item type specific logic
	if req.ItemType == "SRV" {
		product.PurchasePrice = 0
		product.StockQuantity = 0
		product.MinStockLimit = 0
	} else {
		if req.PurchasePrice != nil {
			product.PurchasePrice = *req.PurchasePrice
		}
		if req.StockQuantity != nil {
			product.StockQuantity = *req.StockQuantity
		}
		if req.MinStockLimit != nil {
			product.MinStockLimit = *req.MinStockLimit
		} else {
			product.MinStockLimit = 5 // default
		}
	}

	return s.repo.Update(ctx, product)
}

// DeleteProduct soft deletes a product
func (s *ProductService) DeleteProduct(ctx context.Context, id int, adminUser string) error {
	return s.repo.SoftDelete(ctx, id, adminUser)
}

// RestockProduct validates and runs restocking transaction
func (s *ProductService) RestockProduct(ctx context.Context, req domain.RestockRequest, creator string) error {
	if req.ProductID <= 0 {
		return errors.New("ID produk tidak valid")
	}
	if req.Quantity <= 0 {
		return errors.New("kuantitas restock harus lebih besar dari 0")
	}
	if req.PurchasePrice <= 0 {
		return errors.New("harga beli restock harus lebih besar dari 0")
	}
	if req.ReferenceID == "" {
		return errors.New("nomor referensi (PO number) wajib diisi")
	}
	return s.repo.RestockProductTx(ctx, req, creator)
}

// GetStockLogs retrieves stock logs list for a product
func (s *ProductService) GetStockLogs(ctx context.Context, prodID int) ([]*domain.StockLog, error) {
	if prodID <= 0 {
		return nil, errors.New("ID produk tidak valid")
	}
	return s.repo.FindStockLogsByProductID(ctx, prodID)
}

// GetAllCategories retrieves all active categories, optionally filtered by itemTypeID
func (s *ProductService) GetAllCategories(ctx context.Context, itemTypeID int) ([]domain.Category, error) {
	return s.repo.FindAllCategories(ctx, itemTypeID)
}

// CreateCategory creates a new category record with itemTypeId
func (s *ProductService) CreateCategory(ctx context.Context, name string, itemTypeId int) (int, error) {
	if name == "" {
		return 0, errors.New("Nama kategori wajib diisi")
	}
	return s.repo.InsertCategory(ctx, name, itemTypeId)
}

// UpdateCategory updates an existing category
func (s *ProductService) UpdateCategory(ctx context.Context, id int, name string, itemTypeId int) error {
	if name == "" {
		return errors.New("Nama kategori wajib diisi")
	}
	return s.repo.UpdateCategory(ctx, id, name, itemTypeId)
}

// DeleteCategory soft deletes a category
func (s *ProductService) DeleteCategory(ctx context.Context, id int) error {
	return s.repo.DeleteCategory(ctx, id)
}


