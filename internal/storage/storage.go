package storage

import (
	"dag-mpt-app/internal/dag"
	"dag-mpt-app/internal/models"
	"dag-mpt-app/internal/mpt"
	"encoding/json"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog"
)

type Storage struct {
	db     *badger.DB
	dag    *dag.DAG
	mpt    *mpt.MPT
	logger zerolog.Logger
}

func NewStorage(db *badger.DB, logger zerolog.Logger) *Storage {
	return &Storage{
		db:     db,
		dag:    dag.NewDAG(db, logger),
		mpt:    mpt.NewMPT(db, logger),
		logger: logger,
	}
}

func (s *Storage) SaveTransaction(tx models.Transaction) (string, error) {
	txID, err := s.dag.AddTransaction(tx)
	if err != nil {
		return "", err
	}
	s.logger.Info().Str("tx_id", txID).Msg("Transaction saved in DAG")
	return txID, nil
}

func (s *Storage) GetTransaction(id string) (models.Transaction, error) {
	tx, _, err := s.dag.GetTransaction(id)
	if err != nil {
		return models.Transaction{}, err
	}
	return tx, nil
}

func (s *Storage) GetAllTransactions(page, limit int) ([]models.Transaction, int, error) {
	var txs []models.Transaction
	var total int

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		iter := txn.NewIterator(opts)
		defer iter.Close()

		prefix := []byte("tx:")
		// Count total transactions
		for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
			total++
		}

		// Calculate offset and fetch paginated transactions
		offset := (page - 1) * limit
		current := 0
		iter.Seek(prefix)
		for iter.ValidForPrefix(prefix) && len(txs) < limit {
			if current >= offset {
				item := iter.Item()
				var tx models.Transaction
				err := item.Value(func(val []byte) error {
					return json.Unmarshal(val, &tx)
				})
				if err != nil {
					return err
				}
				txs = append(txs, tx)
			}
			current++
			iter.Next()
		}
		return nil
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to retrieve all transactions")
		return nil, 0, err
	}
	s.logger.Info().Int("count", len(txs)).Int("page", page).Int("limit", limit).Msg("Paginated transactions retrieved")
	return txs, total, nil
}

func (s *Storage) DeleteTransaction(id string) error {
	if err := s.dag.DeleteTransaction(id); err != nil {
		return err
	}
	s.logger.Info().Str("tx_id", id).Msg("Transaction deleted from DAG")
	return nil
}

func (s *Storage) SaveAccountState(state models.AccountState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err := s.mpt.Insert(state.Address, data); err != nil {
		return err
	}
	s.logger.Info().Str("address", state.Address).Msg("Account state saved in MPT")
	return nil
}
