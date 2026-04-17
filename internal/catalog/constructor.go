package catalog

import "github.com/hashicorp/go-memdb"

func NewCatalog() Catalog {
	return NewInMemoryCatalog()
}

func NewInMemoryCatalog() *InMemoryCatalog {
	db, err := memdb.NewMemDB(newCatalogSchema())
	if err != nil {
		panic(err)
	}
	return &InMemoryCatalog{db: db}
}
