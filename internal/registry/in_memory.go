package registry

import (
	"sync"

	"github.com/hashicorp/go-memdb"
	"go.uber.org/zap"
)

type InMemoryRegistry struct {
	internal *memdb.MemDB
	mu       sync.RWMutex
	logger   *zap.SugaredLogger
}

func NewInMemoryRegistry(logger *zap.SugaredLogger) (*InMemoryRegistry, error) {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"entry": {
				Name: "entry",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "RequestPath"},
					},
					"actual": {
						Name:    "actual",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "ActualPath"},
					},
					"mime": {
						Name:    "mime",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "MimeType"},
					},
					"version": {
						Name:    "version",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Version"},
					},
				},
			},
		},
	}
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}
	return &InMemoryRegistry{internal: db, logger: logger}, nil
}
func (r *InMemoryRegistry) Register(_ string, entry *Entry) error {
	r.logger.Debugf("Register%v", entry)
	return r.withTxn(true, func(txn *memdb.Txn) error {
		return txn.Insert("entry", entry)
	})
}

func (r *InMemoryRegistry) Lookup(path string) (*Entry, bool) {
	var result *Entry
	_ = r.withTxn(false, func(txn *memdb.Txn) error {
		raw, err := txn.First("entry", "id", path)
		if err != nil || raw == nil {
			return nil
		}
		result, _ = raw.(*Entry)
		return nil
	})
	return result, result != nil
}

func (r *InMemoryRegistry) List() map[string]*Entry {
	result := make(map[string]*Entry)
	_ = r.withTxn(false, func(txn *memdb.Txn) error {
		it, err := txn.Get("entry", "id")
		if err != nil {
			return err
		}
		for obj := it.Next(); obj != nil; obj = it.Next() {
			if entry, ok := obj.(*Entry); ok {
				result[entry.RequestPath] = entry
			}
		}
		return nil
	})
	return result
}

// LookupByMime 按 mimeType 查询
func (r *InMemoryRegistry) LookupByMime(mime string) []*Entry {
	var results []*Entry
	_ = r.withTxn(false, func(txn *memdb.Txn) error {
		it, err := txn.Get("entry", "mime", mime)
		if err != nil {
			return err
		}
		for obj := it.Next(); obj != nil; obj = it.Next() {
			if entry, ok := obj.(*Entry); ok {
				results = append(results, entry)
			}
		}
		return nil
	})
	return results
}

func (r *InMemoryRegistry) withTxn(write bool, fn func(txn *memdb.Txn) error) error {
	if write {
		r.mu.Lock()
		defer r.mu.Unlock()
	} else {
		r.mu.RLock()
		defer r.mu.RUnlock()
	}
	txn := r.internal.Txn(write)
	defer txn.Abort()

	if err := fn(txn); err != nil {
		return err
	}
	if write {
		txn.Commit()
	}
	return nil
}
