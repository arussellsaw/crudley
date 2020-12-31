# Crudley

Crudley is a Go library for building CRUD REST APIs on top of document stores. The goal of the project is to make it less labour intensive to build simple APIs, and allow developers to focus on building important stuff. 

The project works by using a core `crudley.Model` interface, which provides the library with the basic functions needed to create, read, update, and delete data. The stores are also implemented via a `crudley.Store` interface, which means the database backend can easily be swapped by creating a new `crudley.Store` interface. 

There are a couple of different places where the functionality of crudley, but generally i'd recommend writing bespoke API handlers if you want advanced functionality.

* Hooks, these are functions called by the handlers at certain points in the request lifecycle, at the moment only `Hooks.Authorize` is supported for checking authorization.

* Interfaces, pretty well everything in crudley is an interface, so many components can be extended by wrapping and embedding interfaces, overriding specific functions you want to expand on.


```go

// a little example for using this framework
// some example requests:
//
// create a record
// curl -XPOST localhost:3000/api/cool-model/ -d '{"name":"toot-toot"}'
//
// get a list of all records
// curl localhost:3000/api/cool-model/ |jq
//
// get a record by ID
// curl localhost:3000/api/cool-model/{some_id} | jq
//
// get a record by name
// curl 'localhost:3000/api/cool-model/?name=toot-toot' |jq
//
// rename a record
// curl -XPUT localhost:3000/api/cool-model/{some_id} -d '{"name":"foobar"}'

package main

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/arussellsaw/crudley"
	"github.com/arussellsaw/crudley/stores/mem"
)

type MyCoolModel struct {
    // crudley.Base implements most standard fields of the interface for you
    // if you want more control or different behaviour you can implement
    // your own Base model
    crudley.Base
    Name string
    Count int `rest:"immutable"` 
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
    s := mem.NewStore()
    r.PathPrefix("/api/cool-model/").
        Handler(
            http.StripPrefix("/api/cool-model", 
            crudley.NewPath(&MyCoolModel{}, s),
        ),
    )
    
    http.ListenAndServe(":3000", r)
}
```