package main

import (
	"context"
	"github.com/arussellsaw/crudley"
	"github.com/arussellsaw/crudley/stores/firestore"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

// a little example for using this framework with the firestore backend
// some example requests:
// note: you'll need gcloud SDK set up and pointed at
// the incdnt project for this to work
//
// get a list of all incidents
// curl localhost:3000/incidents |jq
//
// get an incident by ID
// curl localhost:3000/incidents/incident_1320757205649702912 | jq
//
// get an incident by name
// curl 'localhost:3000/incidents?name=toot-toot' |jq
//
// rename an incident
// curl -XPUT localhost:3000/incidents/incident_1320757205649702912 -d '{"name":"foobar"}'

func main() {
	ctx := context.Background()

	// first we create a crudley.Store firestore instance
	s, err := firestore.NewStore(ctx, "incdnt")
	if err != nil {
		log.Fatal(err)
	}

	// next we create the path for handling all of the REST
	// requests for this model, it will register PUT, POST, GET, DELETE
	// handlers for CRUDing this object
	r := mux.NewRouter()
	incidentPath := crudley.Path{
		Model:  &Incident{},
		Store:  s,
		Router: r,
		Permit: []string{"GET", "POST", "PUT", "DELETE"},
	}

	incidentPath.Register()

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
