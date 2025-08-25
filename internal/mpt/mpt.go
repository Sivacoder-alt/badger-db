package mpt

import (
	"crypto/sha256"
	"encoding/json"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog"
)

type MPTNode struct {
	Key   []byte
	Value []byte
	Nodes [17]*MPTNode 
}

type MPT struct {
	db     *badger.DB
	root   []byte
	logger zerolog.Logger
}

func NewMPT(db *badger.DB, logger zerolog.Logger) *MPT {
	return &MPT{
		db:     db,
		root:   nil,
		logger: logger,
	}
}

func (m *MPT) Insert(key string, value []byte) error {
	return m.db.Update(func(txn *badger.Txn) error {
		var currentRoot *MPTNode
		if m.root != nil {
			item, err := txn.Get(m.root)
			if err != nil {
				return err
			}
			err = item.Value(func(val []byte) error {
				return json.Unmarshal(val, &currentRoot)
			})
			if err != nil {
				return err
			}
		} else {
			currentRoot = &MPTNode{}
		}

		newRoot := m.insertNode(currentRoot, []byte(key), value)
		data, err := json.Marshal(newRoot)
		if err != nil {
			return err
		}

		hash := sha256.Sum256(data)
		m.root = hash[:]
		return txn.Set(hash[:], data)
	})
}

func (m *MPT) insertNode(node *MPTNode, key []byte, value []byte) *MPTNode {
	if node == nil {
		node = &MPTNode{}
	}

	if len(key) == 0 {
		node.Value = value
		return node
	}

	index := key[0]
	if index >= 16 {
		m.logger.Error().Msg("Invalid key index")
		return node
	}

	node.Nodes[index] = m.insertNode(node.Nodes[index], key[1:], value)
	return node
}

func (m *MPT) Get(key string) ([]byte, error) {
	var value []byte
	err := m.db.View(func(txn *badger.Txn) error {
		if m.root == nil {
			return badger.ErrKeyNotFound
		}

		item, err := txn.Get(m.root)
		if err != nil {
			return err
		}

		var rootNode MPTNode
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &rootNode)
		})
		if err != nil {
			return err
		}

		value = m.getNode(&rootNode, []byte(key))
		if value == nil {
			return badger.ErrKeyNotFound
		}
		return nil
	})
	return value, err
}

func (m *MPT) getNode(node *MPTNode, key []byte) []byte {
	if node == nil {
		return nil
	}

	if len(key) == 0 {
		return node.Value
	}

	index := key[0]
	if index >= 16 {
		return nil
	}

	return m.getNode(node.Nodes[index], key[1:])
}
