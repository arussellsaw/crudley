# rest

A toolkit for building CRUD apis on top of KV / NoSQL backends.

Define Models that satisfy an interface (crudley.Model), and you're ready to go.

```go
package main

import (
	"http"

	"github.com/arussellsaw/crudley"
	"github.com/arussellsaw/crudley/stores/mem"
	"github.com/gorilla/mux"
)

type MyCoolModel struct {
	// crudley.Base implements most standard fields of the interface for you
	// if you want more control or different behaviour you can implement
	// your own Base model
	crudley.Base `bson:",inline"`
	Name string
	Count int
}

// New creates a new instance of your model
func (m *MyCoolModel) New(id string) crudley.Model {
	return &MyCoolModel{Base: crudley.NewBase(id)}
}

// GetName returns the name of your model for database tables/collections
func (m *MyCoolModel) GetName() string {
	return "mycoolmodel"
}

func main() {
	r := mux.NewRouter()
	jobPath := crudley.Path{
		Model: &MyCoolModel{},
		Store: mem.NewStore(),
		Router: r,
		Permit: []string{"GET", "POST", "DELETE"},
	}
	http.ListenAndServe(":3000", r)
}

```

A lot of the concepts were designed around a target of mongodb, but practically any key value store will work well.

