package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"contract-service/internal/model"
	"contract-service/internal/repository"
)

type ContractHandler struct {
	repo *repository.ContractRepository
	log  *slog.Logger
}

func New(repo *repository.ContractRepository, log *slog.Logger) *ContractHandler {
	return &ContractHandler{repo: repo, log: log}
}

// CreateTimeSlice handles POST /time-slices
func (h *ContractHandler) CreateTimeSlice(w http.ResponseWriter, r *http.Request) {
	var payload model.IncomingPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.log.Warn("failed to decode payload", "err", err)
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}

	if err := validatePayload(&payload); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	id, err := h.repo.CreateTimeSlice(r.Context(), &payload)
	if err != nil {
		if errors.Is(err, model.ErrAllArticlesKnown) {
			writeError(w, http.StatusConflict,
				"all provided article IDs already exist for this contract — no new time slice created")
			return
		}
		h.log.Error("failed to create time slice", "err", err, "contract_id", payload.ContractID)
		writeError(w, http.StatusInternalServerError, "could not persist time slice")
		return
	}

	h.log.Info("time slice created", "id", id, "contract_id", payload.ContractID)
	writeJSON(w, http.StatusCreated, model.SuccessResponse{
		Message:     "time slice created",
		TimeSliceID: id,
	})
}

// GetTimeSlices handles GET /time-slices/{contract_id}
func (h *ContractHandler) GetTimeSlices(w http.ResponseWriter, r *http.Request) {
	// Extract contract_id from path: /time-slices/{contract_id}
	contractID := strings.TrimPrefix(r.URL.Path, "/time-slices/")
	if contractID == "" {
		writeError(w, http.StatusBadRequest, "contract_id is required in the URL path")
		return
	}

	slices, err := h.repo.GetTimeSlicesByContract(r.Context(), contractID)
	if err != nil {
		h.log.Error("failed to fetch time slices", "err", err, "contract_id", contractID)
		writeError(w, http.StatusInternalServerError, "could not fetch time slices")
		return
	}

	if len(slices) == 0 {
		writeError(w, http.StatusNotFound, "no time slices found for contract_id: "+contractID)
		return
	}

	writeJSON(w, http.StatusOK, slices)
}

func validatePayload(p *model.IncomingPayload) error {
	if p.ContractID == "" {
		return validationError{"contract_id is required"}
	}
	if len(p.ArticleIDs) == 0 {
		return validationError{"article_ids must not be empty"}
	}
	if p.ValidityTag == "" {
		return validationError{"validity_tag is required"}
	}
	if p.InvoiceDate.IsZero() {
		return validationError{"invoice_date is required"}
	}
	return nil
}

type validationError struct{ msg string }

func (e validationError) Error() string { return e.msg }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, model.ErrorResponse{Error: msg})
}
