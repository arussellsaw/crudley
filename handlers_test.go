package crudley_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/arussellsaw/crudley"
	"github.com/arussellsaw/crudley/stores/mem"
	"github.com/arussellsaw/crudley/testutil/model"
)

var client = http.Client{}

func setUpTestPath() (*mux.Router, *crudley.Path, error) {
	store := mem.NewStore()
	col, err := store.Collection(&model.TestModel{})
	if err != nil {
		return nil, nil, err
	}
	var testModels = []*model.TestModel{
		&model.TestModel{
			StringVal: "model1",
			IntVal:    1,
			Owner: 	   "foo",
		},
		&model.TestModel{
			StringVal: "model2",
			IntVal:    2,
			StructVal: model.StructVal{
				"foo",
			},
			Owner: 	   "foo",
		},
		&model.TestModel{
			StringVal: "model3",
			IntVal:    3,
			Owner: 	   "foo",
		},
		&model.TestModel{
			StringVal: "model4",
			IntVal:    4,
			Owner: 	   "bar",
		},
		&model.TestModel{
			StringVal: "model5",
			IntVal:    5,
			Owner: 	   "bar",
		},
		&model.TestModel{
			StringVal: "model6",
			IntVal:    6,
			Owner: 	   "bar",
		},
	}
	for _, mdl := range testModels {
		err := col.Create(context.Background(), func(id string) (crudley.Model, error) {
			mdl.ID = id
			return mdl, nil
		})
		if err != nil {
			return nil, nil, err
		}
	}
	r := mux.NewRouter()
	p := crudley.NewPath(&model.TestModel{}, store)
	r.PathPrefix("/api/test/").Handler(http.StripPrefix("/api/test", p))

	return r, p, nil
}

func testHandler(method, URL string, body io.Reader) (model.TestModelResponse, error) {
	tmr := model.TestModelResponse{}
	r, err := http.NewRequest(method, URL, body)
	res, err := client.Do(r)
	if err != nil {
		return tmr, err
	}
	tmr.RawResponse, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return tmr, err
	}
	err = json.Unmarshal(tmr.RawResponse, &tmr)
	return tmr, err
}

func TestGET(t *testing.T) {
	r, _, err := setUpTestPath()
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	s := httptest.NewServer(r)
	defer s.Close()
	tmr, err := testHandler("GET", fmt.Sprintf("%s/api/test/", s.URL), nil)
	if err != nil {
		t.Errorf("expected nil, got %s - %s", err, string(tmr.RawResponse))
	}
	if len(tmr.Models) != 6 {
		t.Errorf("expected 6, got %v", len(tmr.Models))
	}
	tmr, err = testHandler("GET", fmt.Sprintf("%s/api/test/?string_val=model6", s.URL), nil)
	if err != nil {
		t.Errorf("expected nil, got %s - %s", err, string(tmr.RawResponse))
	}
	if len(tmr.Models) != 1 {
		t.Errorf("expected 1, got %v", len(tmr.Models))
	}
	if tmr.Models[0].StringVal != "model6" {
		t.Errorf("expected model6, got %s", tmr.Models[0].StringVal)
	}
}

func TestGETID(t *testing.T) {
	r, path, err := setUpTestPath()
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	col, err := path.Store.Collection(&model.TestModel{})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	var mID string
	err = col.Create(context.Background(), func(id string) (crudley.Model, error) {
		mID = id
		return &model.TestModel{ID: id, StringVal: "newmodel", StructVal: model.StructVal{Field: "foo"}}, nil
	})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.URL.String())
	})
	s := httptest.NewServer(r)
	defer s.Close()
	tmr, err := testHandler("GET", fmt.Sprintf("%s/api/test/%s", s.URL, mID), nil)
	if err != nil {
		t.Errorf("expected nil, got %s - %s", err, string(tmr.RawResponse))
	}
	if len(tmr.Models) != 1 {
		t.Errorf("expected 1, got %v", len(tmr.Models))
	}
	if tmr.Models[0].ID != mID {
		t.Errorf("expected %s, got %s", mID, tmr.Models[0].ID)
	}
	if tmr.Models[0].StringVal != "newmodel" {
		t.Errorf("expected newmodel, got %s", tmr.Models[0].StringVal)
	}
	if tmr.Models[0].StructVal.Field != "foo" {
		t.Errorf("expected newmodel, got %s", tmr.Models[0].StringVal)
	}
}

func TestPOST(t *testing.T) {
	r, path, err := setUpTestPath()
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	s := httptest.NewServer(r)
	defer s.Close()
	modelBuf := `{"string_val": "test1", "int_val":3}`
	tmr, err := testHandler("POST", fmt.Sprintf("%s/api/test/", s.URL), bytes.NewBufferString(modelBuf))
	if err != nil {
		t.Errorf("expected nil, got %s - %s", err, string(tmr.RawResponse))
	}
	if len(tmr.Models) != 1 {
		t.Errorf("expected 1, got %v", len(tmr.Models))
	}
	id := tmr.Models[0].ID
	col, err := path.Store.Collection(&model.TestModel{})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	mdl, err := col.View(context.Background(), id)
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
	r, path, err := setUpTestPath()
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	col, err := path.Store.Collection(&model.TestModel{})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	var mID string
	err = col.Create(context.Background(), func(id string) (crudley.Model, error) {
		mID = id
		return &model.TestModel{ID: id, StringVal: "newmodel", IntVal: 45}, nil
	})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	s := httptest.NewServer(r)
	defer s.Close()
	modelBuf := `{"string_val": "updatedmodel"}`
	_, err = testHandler("PUT", fmt.Sprintf("%s/api/test/%s", s.URL, mID), bytes.NewBufferString(modelBuf))
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	mdl, err := col.View(context.Background(), mID)
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
	r, path, err := setUpTestPath()
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	col, err := path.Store.Collection(&model.TestModel{})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	var mID string
	err = col.Create(context.Background(), func(id string) (crudley.Model, error) {
		mID = id
		return &model.TestModel{ID: id, StringVal: "newmodel"}, nil
	})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	s := httptest.NewServer(r)
	defer s.Close()
	_, err = testHandler("DELETE", fmt.Sprintf("%s/api/test/%s", s.URL, mID), nil)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	mdl, err := col.View(context.Background(), mID)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	if mdl != nil {
		t.Errorf("expected nil, got %v",mdl)
	}
}

func TestAuthoriseGET(t *testing.T) {
	model.AuthoriseFunc = func(ctx context.Context, action crudley.Action, m *model.TestModel) error {
		// empty searches become filtered to owner
		if m.Owner == "" {
			m.Owner = "foo"
		}
		if m.Owner != "foo" {
			// can't list for owners other than yourself
			return fmt.Errorf("unauthorised!")
		}
		return nil
	}
	defer func() {model.AuthoriseFunc = nil}()
	r, _, err := setUpTestPath()
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	s := httptest.NewServer(r)
	defer s.Close()

	// list using empty query, Authorise should append query params to prevent listing everything
	tmr, err := testHandler("GET", fmt.Sprintf("%s/api/test/", s.URL), nil)
	if err != nil {
		t.Errorf("expected nil, got %s - %s", err, string(tmr.RawResponse))
	}
	if len(tmr.Models) != 3 {
		t.Errorf("expected 3, got %v", len(tmr.Models))
	}
	tmr, err = testHandler("GET", fmt.Sprintf("%s/api/test/?string_val=model3", s.URL), nil)
	if err != nil {
		t.Errorf("expected nil, got %s - %s", err, string(tmr.RawResponse))
	}
	if len(tmr.Models) != 1 {
		t.Errorf("expected 1, got %v", len(tmr.Models))
		return
	}
	if tmr.Models[0].StringVal != "model3" {
		t.Errorf("expected model3, got %s", tmr.Models[0].StringVal)
	}

	// try and get list for models we don't own
	tmr, err = testHandler("GET", fmt.Sprintf("%s/api/test/?owner=bar", s.URL), nil)
	if err != nil {
		t.Errorf("expected nil, got %s - %s", err, string(tmr.RawResponse))
	}
	if len(tmr.Errors) == 0 {
		t.Errorf("expected some errors, got none")
	}
	if len(tmr.Models) != 0 {
		t.Errorf("expected 0, got %v", len(tmr.Models))
		return
	}
}
