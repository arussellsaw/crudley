package crudley_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/arussellsaw/crudley"
	"github.com/arussellsaw/crudley/stores/mem"
	"github.com/arussellsaw/crudley/testutil/model"
)

var client = http.Client{}

func setUpTestPath() (*rest.Path, error) {
	store := mem.NewStore()
	col, err := store.Collection(&model.TestModel{})
	if err != nil {
		return nil, err
	}
	var testModels = []*model.TestModel{
		&model.TestModel{
			StringVal: "model1",
			IntVal:    1,
		},
		&model.TestModel{
			StringVal: "model2",
			IntVal:    2,
		},
		&model.TestModel{
			StringVal: "model3",
			IntVal:    3,
		},
		&model.TestModel{
			StringVal: "model4",
			IntVal:    4,
		},
		&model.TestModel{
			StringVal: "model5",
			IntVal:    5,
		},
		&model.TestModel{
			StringVal: "model6",
			IntVal:    6,
		},
	}
	for _, mdl := range testModels {
		err := col.Create(func(id string) (rest.Model, error) {
			mdl.ID = id
			return mdl, nil
		})
		if err != nil {
			return nil, err
		}
	}
	r := mux.NewRouter()
	p := &rest.Path{
		Store:           store,
		Model:           &model.TestModel{},
		Permit:          []string{"GET", "POST", "PUT", "DELETE"},
		Router:          r,
		MarshalResponse: model.MarshalTestModelResponse,
	}
	p.Register()
	return p, nil
}

func testHandler(method, URL string, body io.Reader) (*model.TestModelResponse, error) {
	r, err := http.NewRequest(method, URL, body)
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	tmr := &model.TestModelResponse{}
	err = json.NewDecoder(res.Body).Decode(tmr)
	return tmr, err
}

func TestGET(t *testing.T) {
	path, err := setUpTestPath()
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	s := httptest.NewServer(path.Router)
	defer s.Close()
	tmr, err := testHandler("GET", fmt.Sprintf("%s/testmodel", s.URL), nil)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	if len(tmr.Results) != 6 {
		t.Errorf("expected 6, got %v", len(tmr.Results))
	}
	tmr, err = testHandler("GET", fmt.Sprintf("%s/testmodel?string_val=model6", s.URL), nil)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	if len(tmr.Results) != 1 {
		t.Errorf("expected 1, got %v", len(tmr.Results))
	}
	if tmr.Results[0].StringVal != "model6" {
		t.Errorf("expected model6, got %s", tmr.Results[0].StringVal)
	}
}

func TestGETID(t *testing.T) {
	path, err := setUpTestPath()
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	col, err := path.Store.Collection(&model.TestModel{})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	var mID string
	err = col.Create(func(id string) (rest.Model, error) {
		mID = id
		return &model.TestModel{ID: id, StringVal: "newmodel"}, nil
	})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	s := httptest.NewServer(path.Router)
	defer s.Close()
	tmr, err := testHandler("GET", fmt.Sprintf("%s/testmodel/%s", s.URL, mID), nil)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	if len(tmr.Results) != 1 {
		t.Errorf("expected 1, got %v", len(tmr.Results))
	}
	if tmr.Results[0].ID != mID {
		t.Errorf("expected %s, got %s", mID, tmr.Results[0].ID)
	}
	if tmr.Results[0].StringVal != "newmodel" {
		t.Errorf("expected newmodel, got %s", tmr.Results[0].StringVal)
	}
}

func TestPOST(t *testing.T) {
	path, err := setUpTestPath()
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	s := httptest.NewServer(path.Router)
	defer s.Close()
	modelBuf := `{"string_val": "test1", "int_val":3}`
	tmr, err := testHandler("POST", fmt.Sprintf("%s/testmodel", s.URL), bytes.NewBufferString(modelBuf))
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	if len(tmr.Results) != 1 {
		t.Errorf("expected 1, got %v", len(tmr.Results))
	}
	id := tmr.Results[0].ID
	col, err := path.Store.Collection(&model.TestModel{})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	mdl, err := col.View(id)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	tmdl, ok := mdl.(*model.TestModel)
	if !ok {
		t.Errorf("expected *model.TestModel, got %T", mdl)
	}
	if tmdl.StringVal != "test1" {
		t.Errorf("expected test1, got %s", tmdl.StringVal)
	}
	if tmdl.IntVal != 3 {
		t.Errorf("expected 3, got %v", tmdl.IntVal)
	}
}

func TestPUT(t *testing.T) {
	path, err := setUpTestPath()
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	col, err := path.Store.Collection(&model.TestModel{})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	var mID string
	err = col.Create(func(id string) (rest.Model, error) {
		mID = id
		return &model.TestModel{ID: id, StringVal: "newmodel", IntVal: 45}, nil
	})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	s := httptest.NewServer(path.Router)
	defer s.Close()
	modelBuf := `{"string_val": "updatedmodel"}`
	_, err = testHandler("PUT", fmt.Sprintf("%s/testmodel/%s", s.URL, mID), bytes.NewBufferString(modelBuf))
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	mdl, err := col.View(mID)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	tmdl, ok := mdl.(*model.TestModel)
	if !ok {
		t.Errorf("expected *model.TestModel, got %T", mdl)
	}
	if tmdl.IntVal != 45 {
		t.Errorf("expected 45, got %v", tmdl.IntVal)
	}
	if tmdl.StringVal != "updatedmodel" {
		t.Errorf("expected updatedmodel, got %s", tmdl.StringVal)
	}
}

func TestDELETE(t *testing.T) {
	path, err := setUpTestPath()
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	col, err := path.Store.Collection(&model.TestModel{})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	var mID string
	err = col.Create(func(id string) (rest.Model, error) {
		mID = id
		return &model.TestModel{ID: id, StringVal: "newmodel"}, nil
	})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	s := httptest.NewServer(path.Router)
	defer s.Close()
	_, err = testHandler("DELETE", fmt.Sprintf("%s/testmodel/%s", s.URL, mID), nil)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	mdl, err := col.View(mID)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	if !mdl.IsDeleted() {
		t.Errorf("expected true, got false")
	}
}
