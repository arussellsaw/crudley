package crudley_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/arussellsaw/crudley"
	"github.com/arussellsaw/crudley/stores/mem"
	"github.com/arussellsaw/crudley/testutil/model"
)

var knownTime = time.Unix(1417711554, 0).UTC()

func TestUnmarshalQuery(t *testing.T) {
	timestr, err := json.Marshal(knownTime)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	URL := fmt.Sprintf("http://example.com/test?string_val=testing&time_val=%s&int_val=30&bool_val=true", timestr[1:len(timestr)-1])
	m := &model.TestModel{}
	r, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	err = rest.UnmarshalQuery(r, m)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	tm := &model.TestModel{
		StringVal: "testing",
		IntVal:    30,
		TimeVal:   knownTime,
		BoolVal:   rest.TruePtr(),
	}
	if !reflect.DeepEqual(m, tm) {
		mout, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			t.Errorf("expected nil, got %s", err)
		}
		tmout, err := json.MarshalIndent(tm, "", "  ")
		if err != nil {
			t.Errorf("expected nil, got %s", err)
		}
		t.Errorf("expected: \n %s \n got: \n %s", string(tmout), string(mout))
	}
}

func TestUnmarshalMultiQuery(t *testing.T) {
	timestr, err := json.Marshal(knownTime)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	URL := fmt.Sprintf("http://example.com/test?string_val=testing&string_val=test2&time_val=%s&int_val=30&bool_val=true", timestr[1:len(timestr)-1])
	m := &model.TestModel{}
	r, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	mdls, err := rest.UnmarshalMultiQuery(r, m)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	if len(mdls) != 2 {
		t.Errorf("expected 2 got %v", len(mdls))
	}
	m1 := &model.TestModel{
		StringVal: "testing",
		IntVal:    30,
		TimeVal:   knownTime,
		BoolVal:   rest.TruePtr(),
	}
	m2 := &model.TestModel{
		StringVal: "test2",
	}
	if !reflect.DeepEqual(mdls[0], m1) {
		mout, err := json.MarshalIndent(mdls[0], "", "  ")
		if err != nil {
			t.Errorf("expected nil, got %s", err)
		}
		tmout, err := json.MarshalIndent(m1, "", "  ")
		if err != nil {
			t.Errorf("expected nil, got %s", err)
		}
		t.Errorf("expected: \n %s \n got: \n %s", string(tmout), string(mout))
	}
	if !reflect.DeepEqual(mdls[1], m2) {
		mout, err := json.MarshalIndent(mdls[1], "", "  ")
		if err != nil {
			t.Errorf("expected nil, got %s", err)
		}
		tmout, err := json.MarshalIndent(m2, "", "  ")
		if err != nil {
			t.Errorf("expected nil, got %s", err)
		}
		t.Errorf("expected: \n %s \n got: \n %s", string(tmout), string(mout))
	}
}

func TestUnmarshalGetQuery(t *testing.T) {
	s := mem.NewStore()
	col, err := s.Collection(&model.TestModel{})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	err = col.Create(func(id string) (rest.Model, error) {
		m := &model.TestModel{
			ID:        id,
			StringVal: "model1",
			IntVal:    5,
			BoolVal:   rest.TruePtr(),
		}
		return m, nil
	})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	err = col.Create(func(id string) (rest.Model, error) {
		m := &model.TestModel{
			ID:        id,
			StringVal: "model2",
			IntVal:    10,
		}
		return m, nil
	})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	r, err := http.NewRequest("GET", "http://example.com/testmodel?int_val_greaterthan=7", nil)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	q := col.Query()
	err = rest.UnmarshalGetQuery(r, &model.TestModel{}, q)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	mdls, err := q.Execute()
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	if len(mdls) != 1 {
		t.Errorf("expected 1, got %v", len(mdls))
	}
	tm, ok := mdls[0].(*model.TestModel)
	if !ok {
		t.Errorf("expected ok, got !ok")
	}
	if tm.IntVal != 10 {
		t.Errorf("expected 10, got %v", tm.IntVal)
	}
	if tm.StringVal != "model2" {
		t.Errorf("expected model2, got %s", tm.StringVal)
	}

	r, err = http.NewRequest("GET", "http://example.com/testmodel?has=bool_val", nil)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	q = col.Query()
	err = rest.UnmarshalGetQuery(r, &model.TestModel{}, q)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	mdls, err = q.Execute()
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	if len(mdls) != 1 {
		t.Errorf("expected 1, got %v", len(mdls))
	}
	tm, ok = mdls[0].(*model.TestModel)
	if !ok {
		t.Errorf("expected ok, got !ok")
	}
	if tm.IntVal != 5 {
		t.Errorf("expected 5, got %v", tm.IntVal)
	}
	if tm.StringVal != "model1" {
		t.Errorf("expected model1, got %s", tm.StringVal)
	}

	r, err = http.NewRequest("GET", "http://example.com/testmodel?string_val=model2", nil)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	q = col.Query()
	err = rest.UnmarshalGetQuery(r, &model.TestModel{}, q)
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	mdls, err = q.Execute()
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	if len(mdls) != 1 {
		t.Errorf("expected 1, got %v", len(mdls))
	}
	tm, ok = mdls[0].(*model.TestModel)
	if !ok {
		t.Errorf("expected ok, got !ok")
	}
	if tm.StringVal != "model2" {
		t.Errorf("expected model2, got %s", tm.StringVal)
	}
}
