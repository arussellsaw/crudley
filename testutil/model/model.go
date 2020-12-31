package model

import (
	"time"

	"github.com/arussellsaw/crudley"
)

// TestModel is a testing implementation of the Model interface
type TestModel struct {
	ID        string    `json:"id" bson:"id,omitempty" rest:"immutable"`
	StringVal string    `json:"string_val" bson:"string_val,omitempty"`
	TimeVal   time.Time `json:"time_val" bson:"time_val,omitempty"`
	IntVal    int       `json:"int_val" bson:"int_val,omitempty"`
	BoolVal   *bool     `json:"bool_val" bson:"bool_val,omitempty"`
	Deleted   bool      `json:"deleted" bson:"deleted,omitempty"`
	StructVal StructVal `json:"struct_val"`
}

type StructVal struct {
	Field string `json:"field"`
}

// New creates a new TestModel with the ID set
func (m *TestModel) New(id string) crudley.Model {
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
	RawResponse []byte
	Models      []*TestModel
	Errors      []string
}
