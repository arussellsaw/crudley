package main

import (
	"github.com/arussellsaw/crudley"
	"github.com/arussellsaw/crudley/stores/firestore"
	"github.com/arussellsaw/crudley/stores/mem"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

// a little example for using this framework
// some example requests:
//
// get a list of all incidents
// curl localhost:3000/incident |jq
//
// get an incident by ID
// curl localhost:3000/incident/incident_1320757205649702912 | jq
//
// get an incident by name
// curl 'localhost:3000/incident?name=toot-toot' |jq
//
// rename an incident
// curl -XPUT localhost:3000/incident/incident_1320757205649702912 -d '{"name":"foobar"}'

func main() {

	// first we create a crudley.Store
	s := mem.NewStore()

	// next we create the path for handling all of the REST
	// handlers for CRUDing this object
	r := mux.NewRouter()
	p := crudley.NewPath(&Incident{}, s)

	r.PathPrefix("/incident/").Handler(http.StripPrefix("/incident", p))

	log.Fatal(http.ListenAndServe(":3000", r))
}

// Incident is one of our models, this struct and two methods are all we need to implement
// in order to create a CRUD rest API with per-field searching and querying
type Incident struct {
	Name    string    `json:"name" firestore:"name,omitempty"`
	Closed  time.Time `json:"closed" firestore:"closed,omitempty"`
	Updated time.Time `json:"updated" firestore:"updated,omitempty"`

	// base is a generic implementation of crudley.Model that does things
	// common to all Models, with firestore specific struct tags
	firestore.Base
}

func (i *Incident) New(id string) crudley.Model {
	return &Incident{Base: firestore.Base{ID: id}}
}

func (i *Incident) GetName() string {
	return "incidents"
}
