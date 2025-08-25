package api

import (
	"dag-mpt-app/internal/models"
	"dag-mpt-app/internal/storage"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

type API struct {
	storage *storage.Storage
	logger  zerolog.Logger
}

func NewAPI(storage *storage.Storage, logger zerolog.Logger) *API {
	return &API{storage: storage, logger: logger}
}

func (api *API) writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		api.logger.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

func (api *API) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var tx models.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		api.logger.Error().Err(err).Msg("Failed to decode transaction")
		api.writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid transaction format"})
		return
	}

	tx.ID = ""
	txID, err := api.storage.SaveTransaction(tx)
	if err != nil {
		api.logger.Error().Err(err).Msg("Failed to save transaction -- " + err.Error())
		api.writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save transaction -- " + err.Error()})
		return
	}

	tx.ID = txID
	api.logger.Info().Str("tx_id", tx.ID).Msg("Transaction created")
	api.writeJSONResponse(w, http.StatusCreated, tx)
}

func (api *API) GetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tx, err := api.storage.GetTransaction(vars["id"])
	if err != nil {
		api.logger.Error().Err(err).Str("tx_id", vars["id"]).Msg("Failed to get transaction")
		api.writeJSONResponse(w, http.StatusNotFound, map[string]string{"error": "Failed to get transactions -- " + err.Error()})
		return
	}

	api.logger.Info().Str("tx_id", vars["id"]).Msg("Transaction retrieved")
	api.writeJSONResponse(w, http.StatusOK, tx)
}

func (api *API) GetAllTransactions(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}

	txs, total, err := api.storage.GetAllTransactions(page, limit)
	if err != nil {
		api.logger.Error().Err(err).Msg("Failed to get all transactions")
		api.writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve transactions"})
		return
	}

	totalPages := (total + limit - 1) / limit

	response := map[string]interface{}{
		"transactions": txs,
		"pagination": map[string]int{
			"current_page": page,
			"limit":        limit,
			"total":        total,
			"total_pages":  totalPages,
		},
	}

	api.logger.Info().Int("count", len(txs)).Int("page", page).Int("limit", limit).Msg("All transactions retrieved")
	api.writeJSONResponse(w, http.StatusOK, response)
}

func (api *API) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := api.storage.DeleteTransaction(vars["id"]); err != nil {
		api.logger.Error().Err(err).Str("tx_id", vars["id"]).Msg("Failed to delete transaction")
		api.writeJSONResponse(w, http.StatusNotFound, map[string]string{"error": "Transaction not found" + err.Error()})
		return
	}

	api.logger.Info().Str("tx_id", vars["id"]).Msg("Transaction deleted")
	api.writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Transaction deleted successfully"})
}
