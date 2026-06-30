package repository

import (
	"context"
	"database/sql"
	"pro-autogarage-api/internal/domain"
)

type ParamRepository struct {
	db *sql.DB
}

func NewParamRepository(db *sql.DB) *ParamRepository {
	return &ParamRepository{db: db}
}

// FindByGroup retrieves active params filtered by group_param
func (r *ParamRepository) FindByGroup(ctx context.Context, groupParam string) ([]*domain.Param, error) {
	query := `
		SELECT id, nama_param, kode_param 
		FROM params 
		WHERE group_param = $1 AND status = 'Y'
		ORDER BY id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, groupParam)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var params []*domain.Param
	for rows.Next() {
		var p domain.Param
		if err := rows.Scan(&p.ID, &p.NamaParam, &p.KodeParam); err != nil {
			return nil, err
		}
		params = append(params, &p)
	}
	return params, nil
}
