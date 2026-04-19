package catalog

import "github.com/hashicorp/go-memdb"

type MemDBCatalog struct {
	db *memdb.MemDB
}

type InMemoryCatalog = MemDBCatalog

func NewCatalog() Catalog {
	return NewInMemoryCatalog()
}

func NewInMemoryCatalog() *MemDBCatalog {
	db, err := memdb.NewMemDB(newCatalogSchema())
	if err != nil {
		panic(err)
	}
	return &MemDBCatalog{db: db}
}
