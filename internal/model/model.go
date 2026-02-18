package model

import (
	"errors"
	"time"
)

// IncomingPayload represents the HTTP request body
type IncomingPayload struct {
	ContractID  string    `json:"contract_id"`
	ArticleIDs  []int64   `json:"article_ids"`
	ValidityTag string    `json:"validity_tag"`
	InvoiceDate time.Time `json:"invoice_date"`
}

// TimeSlice represents a contract time slice stored in the DB
type TimeSlice struct {
	ID           int64     `db:"id"`
	ContractID   string    `db:"contract_id"`
	TopArticleID int64     `db:"top_article_id"`
	ValidityTag  string    `db:"validity_tag"`
	InvoiceDate  time.Time `db:"invoice_date"`
	CreatedAt    time.Time `db:"created_at"`
}

// ErrorResponse is a standard JSON error envelope
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse is returned on successful creation
type SuccessResponse struct {
	Message     string `json:"message"`
	TimeSliceID int64  `json:"time_slice_id"`
}

// ErrAllArticlesKnown is returned when every article ID in the payload
// already exists as a top_article_id for the given contract.
var ErrAllArticlesKnown = errors.New("all article IDs are already stored for this contract")
