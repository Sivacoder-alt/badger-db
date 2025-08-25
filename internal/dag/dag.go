package dag

import (
	"crypto/sha256"
	"dag-mpt-app/internal/models"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog"
)

type DAG struct {
	db     *badger.DB
	logger zerolog.Logger
}

func NewDAG(db *badger.DB, logger zerolog.Logger) *DAG {
	return &DAG{
		db:     db,
		logger: logger,
	}
}

func (d *DAG) AddTransaction(tx models.Transaction) (string, error) {
	txForHash := tx
	txForHash.ID = ""
	data, err := json.Marshal(txForHash)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	txID := hex.EncodeToString(hash[:])

	err = d.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte("tx:" + txID))
		if err == nil {
			return errors.New("transaction ID already exists")
		}
		if err != badger.ErrKeyNotFound {
			return err
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	tx.ID = txID

	data, err = json.Marshal(tx)
	if err != nil {
		return "", err
	}

	err = d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set([]byte("tx:"+txID), data); err != nil {
			return err
		}

		for _, parentID := range tx.Parents {
			if err := txn.Set([]byte("parent:"+txID+":"+parentID), []byte{}); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		d.logger.Error().Err(err).Str("tx_id", txID).Msg("Failed to add transaction to DAG")
		return "", err
	}

	d.logger.Info().Str("tx_id", txID).Msg("Transaction added to DAG")
	return txID, nil
}

func (d *DAG) GetTransaction(txID string) (models.Transaction, []string, error) {
	var tx models.Transaction
	var parents []string

	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("tx:" + txID))
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &tx)
		})
		if err != nil {
			return err
		}

		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()
		prefix := []byte("parent:" + txID + ":")
		for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
			key := iter.Item().Key()
			parentID := string(key[len(prefix):])
			parents = append(parents, parentID)
		}
		return nil
	})

	if err != nil {
		d.logger.Error().Err(err).Str("tx_id", txID).Msg("Failed to retrieve transaction")
		return models.Transaction{}, nil, err
	}

	d.logger.Info().Str("tx_id", txID).Msg("Transaction retrieved from DAG")
	return tx, parents, nil
}

func (d *DAG) DeleteTransaction(txID string) error {
	err := d.db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte("tx:" + txID)); err != nil {
			return err
		}

		if err := txn.Delete([]byte("tx:" + txID)); err != nil {
			return err
		}

		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()
		prefix := []byte("parent:" + txID + ":")
		for iter.Seek(prefix); iter.ValidForPrefix(prefix); iter.Next() {
			if err := txn.Delete(iter.Item().Key()); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		d.logger.Error().Err(err).Str("tx_id", txID).Msg("Failed to delete transaction")
		return err
	}

	d.logger.Info().Str("tx_id", txID).Msg("Transaction deleted from DAG")
	return nil
}
