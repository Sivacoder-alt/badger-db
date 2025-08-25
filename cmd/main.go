package main

import (
	"dag-mpt-app/internal/api"
	"dag-mpt-app/internal/storage"
	"dag-mpt-app/pkg/logger"
	"net/http"

	"github.com/dgraph-io/badger/v4"
	"github.com/gorilla/mux"
)

func main() {
	log := logger.NewLogger()

	db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open BadgerDB")
	}
	defer db.Close()

	storage := storage.NewStorage(db, log)

	router := mux.NewRouter()
	apiHandler := api.NewAPI(storage, log)

	router.HandleFunc("/tx", apiHandler.CreateTransaction).Methods("POST")
	router.HandleFunc("/tx", apiHandler.GetAllTransactions).Methods("GET")
	router.HandleFunc("/tx/{id}", apiHandler.GetTransaction).Methods("GET")
	router.HandleFunc("/tx/{id}", apiHandler.DeleteTransaction).Methods("DELETE")

	log.Info().Msg("Starting server on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}
