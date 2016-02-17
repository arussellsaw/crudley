package mem

import (
	"testing"

	"code.avct.io/rest/testutil/store"
)

func TestSetGet(t *testing.T) {
	db := NewStore()
	store.TestSetGet(db, t)
}

func TestScan(t *testing.T) {
	db := NewStore()
	store.TestScan(db, t)
}

func TestUpdate(t *testing.T) {
	db := NewStore()
	store.TestUpdate(db, t)
}

func TestSearch(t *testing.T) {
	db := NewStore()
	store.TestSearch(db, t)
}

func TestQuery(t *testing.T) {
	db := NewStore()
	store.TestQuery(db, t)
}
