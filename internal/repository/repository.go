package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"contract-service/internal/model"
)

type ContractRepository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *ContractRepository {
	return &ContractRepository{db: db}
}

// CreateTimeSlice inserts a new time slice for the given contract.
//
// Article-ID selection: picks the highest ID from the payload that does NOT
// already exist as a top_article_id for this contract_id in the DB.
// If all IDs are already present, returns ErrAllArticlesKnown.
//
// Invoice-date update: if the new invoice date differs (day granularity) from
// the stored one for the same contract_id, all existing rows are updated.
func (r *ContractRepository) CreateTimeSlice(ctx context.Context, p *model.IncomingPayload) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 1. Find all article_ids already stored for this contract
	rows, err := tx.Query(ctx,
		`SELECT top_article_id FROM time_slices WHERE contract_id = $1`,
		p.ContractID,
	)
	if err != nil {
		return 0, err
	}
	known := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		known[id] = true
	}
	rows.Close()

	// 2. Pick the highest article ID that is not yet in the DB
	topArticleID, ok := highestNew(p.ArticleIDs, known)
	if !ok {
		return 0, model.ErrAllArticlesKnown
	}

	// 3. Check if an invoice date already exists and differs
	var existingInvoiceDate *time.Time
	row := tx.QueryRow(ctx,
		`SELECT invoice_date FROM time_slices WHERE contract_id = $1 ORDER BY created_at DESC LIMIT 1`,
		p.ContractID,
	)
	var tmp time.Time
	if err := row.Scan(&tmp); err == nil {
		existingInvoiceDate = &tmp
	} else if err != pgx.ErrNoRows && err != nil {
		return 0, err
	}

	if existingInvoiceDate != nil &&
		!existingInvoiceDate.Truncate(24*time.Hour).Equal(p.InvoiceDate.Truncate(24*time.Hour)) {
		_, err = tx.Exec(ctx,
			`UPDATE time_slices SET invoice_date = $1 WHERE contract_id = $2`,
			p.InvoiceDate, p.ContractID,
		)
		if err != nil {
			return 0, err
		}
	}

	// 4. Insert new time slice
	var newID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO time_slices (contract_id, top_article_id, validity_tag, invoice_date)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		p.ContractID, topArticleID, p.ValidityTag, p.InvoiceDate,
	).Scan(&newID)
	if err != nil {
		return 0, err
	}

	return newID, tx.Commit(ctx)
}

// GetTimeSlicesByContract returns all time slices for a given contract_id,
// ordered by creation date descending (newest first).
func (r *ContractRepository) GetTimeSlicesByContract(ctx context.Context, contractID string) ([]model.TimeSlice, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, contract_id, top_article_id, validity_tag, invoice_date, created_at
		 FROM time_slices
		 WHERE contract_id = $1
		 ORDER BY created_at DESC`,
		contractID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slices []model.TimeSlice
	for rows.Next() {
		var s model.TimeSlice
		if err := rows.Scan(&s.ID, &s.ContractID, &s.TopArticleID, &s.ValidityTag, &s.InvoiceDate, &s.CreatedAt); err != nil {
			return nil, err
		}
		slices = append(slices, s)
	}
	return slices, rows.Err()
}

// highestNew returns the maximum value from ids that is not present in known.
func highestNew(ids []int64, known map[int64]bool) (int64, bool) {
	found := false
	var max int64
	for _, id := range ids {
		if !known[id] && (!found || id > max) {
			max = id
			found = true
		}
	}
	return max, found
}
