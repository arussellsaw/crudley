package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

// ID is the commonly used mux.Var id param.
const ID = "id"

// Path manages building a set of RESTful endpoints for any given Model, using
// the provided Store for a database backend
type Path struct {
	Model Model
	Store Store

	MarshalResponse func(*Response) ([]byte, error)

	// Permit defines the available handlers for the Model
	Permit []string
	// Router is usually passed as a mux.Router.Subrouter() to register the handlers
	// for this Model's routes
	Router *mux.Router

	Middlewares []func(http.Handler) http.Handler

	Collection Collection
	response   Response
}

// Register adds routes for the model
func (p *Path) Register() {
	for _, method := range p.Permit {
		switch method {
		case "GET":
			p.registerPath(fmt.Sprintf("/%s/{id}", p.Model.GetName()), p.Get, "GET")
			p.registerPath(fmt.Sprintf("/%s", p.Model.GetName()), p.Query, "GET")
		case "POST":
			p.registerPath(fmt.Sprintf("/%s", p.Model.GetName()), p.Post, "POST")
		case "PUT":
			p.registerPath(fmt.Sprintf("/%s/{id}", p.Model.GetName()), p.Put, "PUT")
		case "DELETE":
			p.registerPath(fmt.Sprintf("/%s/{id}", p.Model.GetName()), p.Delete, "DELETE")
		}
	}
}

// Handle registers a custom rest.HandlerFunc for the specified path and method
func (p *Path) Handle(path string, h HandlerFunc, method ...string) {
	p.registerPath(path, func(w http.ResponseWriter, r *http.Request) {
		h(p, w, r)
	}, method...)
}

// registerPath registers a http.HandlerFunc for a route, along with the Middlewares specified for this Handler
func (p *Path) registerPath(path string, handler http.HandlerFunc, methods ...string) {
	var cs []alice.Constructor
	cs = append(cs, clearContextMiddleware, p.ResponseMiddleware)
	for _, mw := range p.Middlewares {
		cs = append(cs, mw)
	}
	chain := alice.New(cs...).Then(handler)
	p.Router.Handle(path, chain).Methods(methods...)
}

// InitHandler ensures the collection is initialized for the path, and retrieves
// the response for the request
func (p *Path) InitHandler(w http.ResponseWriter, r *http.Request) (*Response, error) {
	res := GetResponse(r)
	if res == nil {
		http.Error(w, "could not build api response", http.StatusInternalServerError)
		return res, fmt.Errorf("could not retrieve api response")
	}
	var err error

	if p.Collection == nil {
		p.Collection, err = p.Store.Collection(p.Model)
		if err != nil {
			res.AddError(fmt.Errorf("failed to retrieve Collection: %s", err.Error()))
			res.SetStatusCode(http.StatusInternalServerError)
			return res, fmt.Errorf("failed to init collection")
		}
	}
	return res, nil
}

// Query accepts a partial model and looks up the result
func (p *Path) Query(w http.ResponseWriter, r *http.Request) {
	res, err := p.InitHandler(w, r)
	if err != nil {
		return
	}
	out := p.Model.New("")
	q := p.Collection.Query()
	err = UnmarshalGetQuery(r, out, q)
	if err != nil {
		res.AddError(fmt.Errorf("failed to build Query: %s", err.Error()))
		res.SetStatusCode(http.StatusBadRequest)
		return
	}
	models, err := q.Execute()
	if err != nil {
		res.AddError(fmt.Errorf("unexpected error: %s", err.Error()))
		res.SetStatusCode(http.StatusInternalServerError)
		return
	}
	for _, m := range models {
		if !m.IsDeleted() {
			pm, ok := m.(Permissioner)
			if ok {
				if !pm.Permissions(r) {
					continue
				}
			}
			res.AddModel(m)
		}
	}
}

// Get is the http handler for the GET method
func (p *Path) Get(w http.ResponseWriter, r *http.Request) {
	res, err := p.InitHandler(w, r)
	if err != nil {
		return
	}

	vars := mux.Vars(r)
	id, ok := vars[ID]
	if !ok {
		res.AddError(ErrorNoID)
		res.SetStatusCode(http.StatusBadRequest)
		return
	}

	model, err := p.Collection.View(id)
	if err != nil {
		res.AddError(fmt.Errorf("failed to retrieve Model from collection: %s", err.Error()))
		res.SetStatusCode(http.StatusInternalServerError)
		return
	}
	if model == nil {
		res.AddError(ErrorModelNotFound)
		res.SetStatusCode(http.StatusNotFound)
		return
	}

	if p, ok := model.(Permissioner); ok {
		if !p.Permissions(r) {
			res.AddError(ErrorForbidden)
			res.SetStatusCode(http.StatusForbidden)
			return
		}
	}

	if model.IsDeleted() {
		res.AddError(ErrorModelNotFound)
		res.SetStatusCode(http.StatusNotFound)
		return
	}

	res.AddModel(model)
}

// Post saves a Model to the Store
func (p *Path) Post(w http.ResponseWriter, r *http.Request) {
	res, err := p.InitHandler(w, r)
	if err != nil {
		return
	}

	var out Model
	var code = http.StatusOK
	err = p.Collection.Create(func(id string) (Model, error) {
		out = p.Model.New(id)

		err := json.NewDecoder(r.Body).Decode(&RestrictedModel{out})
		if err != nil {
			code = http.StatusInternalServerError
			return out, err
		}

		if p, ok := out.(Permissioner); ok {
			if !p.Permissions(r) {
				code = http.StatusForbidden
				return out, ErrorForbidden
			}
		}
		if v, ok := out.(Validator); ok {
			if err := v.Validate(p.Store); err != nil {
				code = http.StatusBadRequest
				return out, err
			}
		}
		if pre, ok := out.(PreSaver); ok {
			err = pre.PreSave(p.Store)
		}

		return out, err
	})
	if err != nil {
		res.AddError(fmt.Errorf("failed to create Model: %s", err.Error()))
		res.SetStatusCode(http.StatusInternalServerError)
		return
	}

	if post, ok := out.(PostSaver); ok {
		err = post.PostSave(p.Store)
		if err != nil {
			res.AddError(err)
			res.SetStatusCode(http.StatusInternalServerError)
			return
		}
	}
	res.AddModel(out)
}

// Put handles partial JSON to update a Model
func (p *Path) Put(w http.ResponseWriter, r *http.Request) {
	res, err := p.InitHandler(w, r)
	if err != nil {
		return
	}

	vars := mux.Vars(r)
	id, ok := vars[ID]
	if !ok {
		res.AddError(ErrorNoID)
		res.SetStatusCode(http.StatusBadRequest)
		return
	}

	m, err := p.Collection.View(id)
	if err != nil {
		if _, ok := err.(NotFoundError); ok {
			res.AddError(ErrorModelNotFound)
			res.SetStatusCode(http.StatusNotFound)
			return
		}
		res.AddError(fmt.Errorf("failed to retrieve Model from collection: %s", err.Error()))
		res.SetStatusCode(http.StatusInternalServerError)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&RestrictedModel{m})
	if err != nil {
		res.AddError(ErrorMalformedJSON)
		res.SetStatusCode(http.StatusBadRequest)
		return
	}

	if p, ok := m.(Permissioner); ok {
		if !p.Permissions(r) {
			res.AddError(ErrorForbidden)
			res.SetStatusCode(http.StatusForbidden)
			return
		}
	}

	if v, ok := m.(Validator); ok {
		err = v.Validate(p.Store)
		if err != nil {
			res.AddError(ErrorValidationFailed)
			res.SetStatusCode(http.StatusBadRequest)
			return
		}
	}
	if pre, ok := m.(PreSaver); ok {
		err = pre.PreSave(p.Store)
		if err != nil {
			res.AddError(ErrorPreSaveFailed)
			res.SetStatusCode(http.StatusBadRequest)
			return
		}
	}

	err = p.Collection.Update(m.PrimaryKey(), m)
	if err != nil {
		res.AddError(fmt.Errorf("failed to update Model: %s", err.Error()))
		res.SetStatusCode(http.StatusInternalServerError)
		return
	}

	if post, ok := m.(PostSaver); ok {
		err = post.PostSave(p.Store)
		if err != nil {
			res.AddError(ErrorPostSaveFailed)
			res.SetStatusCode(http.StatusBadRequest)
			return
		}
	}

	res.AddModel(m)
}

// Delete handles deleting the Model specified by the mux var "id"
func (p *Path) Delete(w http.ResponseWriter, r *http.Request) {
	res, err := p.InitHandler(w, r)
	if err != nil {
		return
	}

	id, ok := mux.Vars(r)[ID]
	if !ok {
		res.AddError(ErrorNoID)
		res.SetStatusCode(http.StatusBadRequest)
		return
	}

	m, err := p.Collection.View(id)
	if err != nil {
		if _, ok := err.(NotFoundError); ok {
			res.AddError(ErrorModelNotFound)
			return
		}
		res.AddError(fmt.Errorf("failed to retrieve Model: %s", err.Error()))
		res.SetStatusCode(http.StatusInternalServerError)
		return
	}

	if m == nil {
		res.AddError(ErrorModelNotFound)
		return
	}

	if v, ok := m.(Validator); ok {
		err = v.Validate(p.Store)
		if err != nil {
			res.AddError(ErrorValidationFailed)
			res.SetStatusCode(http.StatusBadRequest)
			return
		}
	}

	if p, ok := m.(Permissioner); ok {
		if !p.Permissions(r) {
			res.AddError(ErrorForbidden)
			res.SetStatusCode(http.StatusForbidden)
			return
		}
	}

	m.Delete()

	if pre, ok := m.(PreSaver); ok {
		err = pre.PreSave(p.Store)
		if err != nil {
			res.AddError(ErrorPreSaveFailed)
			res.SetStatusCode(http.StatusBadRequest)
			return
		}
	}

	err = p.Collection.Update(id, m)
	if err != nil {
		res.AddError(fmt.Errorf("failed to update Collection with deleted Model: %s", err.Error()))
		res.SetStatusCode(http.StatusInternalServerError)
		return
	}

	if post, ok := m.(PostSaver); ok {
		err = post.PostSave(p.Store)
		if err != nil {
			res.AddError(ErrorPostSaveFailed)
			res.SetStatusCode(http.StatusBadRequest)
			return
		}
	}
	res.AddModel(m)
}
