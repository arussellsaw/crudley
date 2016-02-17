package model

import (
	"encoding/json"
	"time"

	"github.com/avct/rest"
)

// TestModel is a testing implementation of the Model interface
type TestModel struct {
	ID        string    `json:"id" bson:"id,omitempty" rest:"immutable"`
	StringVal string    `json:"string_val" bson:"string_val,omitempty"`
	TimeVal   time.Time `json:"time_val" bson:"time_val,omitempty"`
	IntVal    int       `json:"int_val" bson:"int_val,omitempty"`
	BoolVal   *bool     `json:"bool_val" bson:"bool_val,omitempty"`
	Deleted   bool      `json:"deleted" bson:"deleted,omitempty"`
}

// New creates a new TestModel with the ID set
func (m *TestModel) New(id string) rest.Model {
	return &TestModel{ID: id}
}

// PrimaryKey returns the TestModel's ID
func (m *TestModel) PrimaryKey() string {
	return m.ID
}

// GetName returns the name of the Model
func (m *TestModel) GetName() string {
	return "testmodel"
}

// Delete marks the TestModel as deleted
func (m *TestModel) Delete() {
	m.Deleted = true
}

// IsDeleted returns the deleted status of the Model
func (m *TestModel) IsDeleted() bool {
	return m.Deleted
}

// TestModelResponse is a response implementation for easy testing of the http
// handlers
type TestModelResponse struct {
	Results []*TestModel
	Errors  []string
}

// MarshalTestModelResponse marshals the rest.Resposne into a TestModelResponse json
func MarshalTestModelResponse(res *rest.Response) ([]byte, error) {
	tmr := &TestModelResponse{}
	for _, mdl := range res.Models {
		tmdl, ok := mdl.(*TestModel)
		if !ok {
			continue
		}
		tmr.Results = append(tmr.Results, tmdl)
	}
	tmr.Errors = res.Errors
	return json.Marshal(tmr)
}
