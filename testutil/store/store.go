package store

import (
	"fmt"
	"testing"

	"github.com/arussellsaw/crudley"
)

// TestableStore is an interface for code generation to create store instances for testing.
type TestableStore interface {
	NewTestingStore() (crudley.Store, error)
	CleanUp()
}

func TestSetGet(store crudley.Store, t *testing.T) {
	var model = &TestModel{}
	col, err := store.Collection(model)
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	var modelID string
	err = col.Create(func(id string) (crudley.Model, error) {
		modelID = id
		md := model.New(id)
		md.(*TestModel).Val = "testing123"
		return md, nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	newModel, err := col.View(modelID)
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	ntm, ok := newModel.(*TestModel)
	if !ok {
		fmt.Printf("%T\n", newModel)
		t.Fatalf("expected true, got false")
	}
	if ntm.Val != "testing123" {
		t.Fatalf("expected testing123, got %s", ntm.Val)
	}
}

func TestScan(store crudley.Store, t *testing.T) {
	var model = &TestModel{}
	col, err := store.Collection(model)
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	var modelID1, modelID2 string
	col.Create(func(id string) (crudley.Model, error) {
		modelID1 = id
		md := model.New(id)
		md.(*TestModel).Val = "testing123"
		return md, nil
	})
	col.Create(func(id string) (crudley.Model, error) {
		modelID2 = id
		md := model.New(id)
		md.(*TestModel).Val = "testing1234"
		return md, nil
	})
	var out = make(map[string]crudley.Model)
	col.Scan(func(mdl crudley.Model) error {
		out[mdl.PrimaryKey()] = mdl
		return nil
	})
	if len(out) != 2 {
		t.Fatalf("expected len 2, got len %v", len(out))
	}
	if mdl, ok := out[modelID1]; !ok {
		t.Fatalf("expected true, got false")
	} else {
		if mdl.(*TestModel).Val != "testing123" {
			t.Fatalf("expected testing123, got %s", mdl.(*TestModel).Val)
		}
	}
	if mdl, ok := out[modelID2]; !ok {
		t.Fatalf("expected true, got false")
	} else {
		if mdl.(*TestModel).Val != "testing1234" {
			t.Fatalf("expected testing1234, got %s", mdl.(*TestModel).Val)
		}
	}
}

func TestUpdate(store crudley.Store, t *testing.T) {
	var model = &TestModel{}
	col, err := store.Collection(model)
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	var modelID string
	col.Create(func(id string) (crudley.Model, error) {
		modelID = id
		md := model.New(id)
		md.(*TestModel).Val = "testing123"
		return md, nil
	})
	newModel, err := col.View(modelID)
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	ntm, ok := newModel.(*TestModel)
	if !ok {
		t.Fatalf("expected true, got false")
	}
	if ntm.Val != "testing123" {
		t.Fatalf("expected testing123, got %s", ntm.Val)
	}
	ntm.Val = "testing1234"
	err = col.Update(modelID, ntm)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	newModel, err = col.View(modelID)
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	ntm, ok = newModel.(*TestModel)
	if !ok {
		t.Fatalf("expected true, got false")
	}
	if ntm.Val != "testing1234" {
		t.Fatalf("expected testing1234, got %s", ntm.Val)
	}
}

func TestSearch(store crudley.Store, t *testing.T) {
	var model = &TestModel{}
	col, err := store.Collection(model)
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	var modelID1, modelID2 string
	col.Create(func(id string) (crudley.Model, error) {
		modelID1 = id
		md := model.New(id)
		md.(*TestModel).Val = "testing123"
		return md, nil
	})
	col.Create(func(id string) (crudley.Model, error) {
		modelID2 = id
		md := model.New(id)
		md.(*TestModel).Val = "testing1234"
		return md, nil
	})
	var out *TestModel
	col.Search(&TestModel{Val: "testing123"}, func(mdl crudley.Model) error {
		out = mdl.(*TestModel)
		return nil
	})
	if out.Val != "testing123" {
		t.Errorf("expected testing123, got %s", out.Val)
	}
}

func TestQuery(store crudley.Store, t *testing.T) {
	var model = &TestModel{}
	col, err := store.Collection(model)
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	var modelID1, modelID2 string
	col.Create(func(id string) (crudley.Model, error) {
		modelID1 = id
		md := model.New(id)
		md.(*TestModel).Val = "testing123"
		md.(*TestModel).Count = 0
		md.(*TestModel).EmbeddedField = "embed"
		return md, nil
	})
	col.Create(func(id string) (crudley.Model, error) {
		modelID2 = id
		md := model.New(id)
		md.(*TestModel).Val = "testing1234"
		md.(*TestModel).Count = 3
		return md, nil
	})
	q := col.Query()
	q.GreaterThan("count", 2)
	res, err := q.Execute()
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1, got %v", len(res))
	}
	if res[0].(*TestModel).Val != "testing1234" {
		t.Errorf("expected testing1234, got %s", err)
	}
	q = col.Query()
	q.Equal("embedded_field", "embed")
	res, err = q.Execute()
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	if len(res) != 1 {
		t.Errorf("expected 1, got %v", len(res))
	}
	if res[0].(*TestModel).Val != "testing123" {
		t.Errorf("expected testing1234, got %s", err)
	}
}

type TestModel struct {
	ID string `json:"_id" bson:"_id,omitempty" rest:"immutable"`

	Embedded `bson:",inline"`

	Val     string `json:"val" bson:"val,omitempty"`
	Count   int    `json:"count" bson:"count,omitempty"`
	Deleted bool   `json:"deleted,omitempty" bson:"deleted,omitempty" rest:"immutable"`
}

type Embedded struct {
	EmbeddedField string `json:"embedded_field" bson:"embedded_field,omitempty"`
}

func (m *TestModel) New(id string) crudley.Model {
	return &TestModel{ID: id}
}

func (m *TestModel) GetName() string {
	return "testmodel"
}

func (m *TestModel) PrimaryKey() string {
	return m.ID
}

func (m *TestModel) Delete() {
	m.Deleted = true
}

func (m *TestModel) IsDeleted() bool {
	return m.Deleted
}
