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

// FindIDByGroupAndCode retrieves the ID of a param by its group and code/name
func (r *ParamRepository) FindIDByGroupAndCode(ctx context.Context, groupParam string, kodeParam string) (int, error) {
	if kodeParam == "" {
		return 0, nil
	}
	query := `
		SELECT id 
		FROM params 
		WHERE group_param = $1 AND (kode_param = $2 OR nama_param = $2 OR id::text = $2) AND status = 'Y' 
		LIMIT 1
	`
	var id int
	err := r.db.QueryRowContext(ctx, query, groupParam, kodeParam).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}
