# rest

A toolkit for building CRUD apis on top of KV / NoSQL backends.

Define Models that satisfy an interface (rest.Model), and you're ready to go.

```go
package main

import (
	"http"

	"github.com/avct/rest"
	"github.com/avct/rest/stores/mem"
	"github.com/gorilla/mux"
)

type MyCoolModel struct {
	// rest.Base implements most standard fields of the interface for you
	// if you want more control or different behaviour you can implement
	// your own Base model
	rest.Base `bson:",inline"`
	Name string
	Count int
}

// New creates a new instance of your model
func (m *MyCoolModel) New(id string) rest.Model {
	return &MyCoolModel{Base: rest.NewBase(id)}
}

// GetName returns the name of your model for database tables/collections
func (m *MyCoolModel) GetName() string {
	return "mycoolmodel"
}

func main() {
	r := mux.NewRouter()
	jobPath := rest.Path{
		Model: &MyCoolModel{},
		Store: mem.NewStore(),
		Router: r,
		Permit: []string{"GET", "POST", "DELETE"},
	}
	http.ListenAndServe(":3000", r)
}

```

A lot of the concepts were designed around a target of mongodb, but practically any key value store will work well.

