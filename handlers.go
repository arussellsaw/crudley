package crudley

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// ID is the commonly used mux.Var id param.
const ID = "id"

func NewPath(m Model, s Store, opt ...Option) *Path {
	p := &Path{
		Model: m,
		Store: s,
	}

	for _, o := range opt {
		o(p)
	}

	r := mux.NewRouter()

	r.Path("/").Methods("GET").HandlerFunc(p.Query)
	r.Path("/{id}").Methods("GET").HandlerFunc(p.Get)

	if !p.ReadOnly {
		r.Path("/").Methods("POST").HandlerFunc(p.Post)
		r.Path("/{id}").Methods("PUT").HandlerFunc(p.Put)
		r.Path("/{id}").Methods("DELETE").HandlerFunc(p.Delete)
	}

	p.r = r

	return p
}

type Option func(p *Path)

func OptionReadOnly(p *Path) {
	p.ReadOnly = true
}

// Path manages building a set of RESTful endpoints for any given Model, using
// the provided Store for a database backend
type Path struct {
	Model Model
	Store Store

	r *mux.Router

	c Collection

	ReadOnly bool
}


func (p *Path) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.r.ServeHTTP(w, r)
}

// InitHandler ensures the collection is initialized for the path, and retrieves
// the response for the request
func (p *Path) initHandler() (Collection, *Response, error) {
	var (
		res = &Response{}
		err error
	)

	c, err := p.Store.Collection(p.Model)
	if err != nil {
		res.AddError(fmt.Errorf("failed to retrieve Collection: %s", err.Error()))
		res.SetStatusCode(http.StatusInternalServerError)
		return nil, nil, fmt.Errorf("failed to init collection")
	}
	return c, res, nil
}

// Query accepts a partial model and looks up the result
func (p *Path) Query(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	c, res, err := p.initHandler()
	if err != nil {
		return
	}
	defer WriteResponse(w, res)
	out := p.Model.New("")

	q := c.Query()
	err = UnmarshalGetQuery(r, out, q)
	if err != nil {
		res.AddError(fmt.Errorf("failed to build Query: %s", err.Error()))
		res.SetStatusCode(http.StatusBadRequest)
		return
	}

	models, err := q.Execute(ctx)
	if err != nil {
		res.AddError(fmt.Errorf("unexpected error: %s", err.Error()))
		res.SetStatusCode(http.StatusInternalServerError)
		return
	}
	for _, m := range models {
		res.AddModel(m)
	}
}

// Get is the http handler for the GET method
func (p *Path) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	c, res, err := p.initHandler()
	if err != nil {
		return
	}
	defer WriteResponse(w, res)

	vars := mux.Vars(r)
	id, ok := vars[ID]
	if !ok {
		res.AddError(ErrorNoID)
		res.SetStatusCode(http.StatusBadRequest)
		return
	}

	model, err := c.View(ctx, id)
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
	if a, ok := model.(Authoriser); ok {
		if err := a.Authorise(ctx, Action{Method: http.MethodGet}); err != nil {
			res.AddError(err)
			res.SetStatusCode(http.StatusNotFound)
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
	ctx := r.Context()
	c, res, err := p.initHandler()
	if err != nil {
		return
	}
	defer WriteResponse(w, res)

	var out Model
	err = c.Create(ctx, func(id string) (Model, error) {
		out = p.Model.New(id)

		err := json.NewDecoder(r.Body).Decode(&RestrictedModel{out})
		if err != nil {
			return out, err
		}

		if a, ok := out.(Authoriser); ok {
			if err := a.Authorise(ctx, Action{Method: http.MethodPost}); err != nil {
				res.SetStatusCode(http.StatusUnauthorized)
				return out, err
			}
		}

		return out, err
	})
	if err != nil {
		res.AddError(fmt.Errorf("failed to create Model: %s", err.Error()))
		if res.GetStatusCode() == 0 {
			res.SetStatusCode(http.StatusInternalServerError)
		}
		return
	}

	res.AddModel(out)
}

// Put handles partial JSON to update a Model
func (p *Path) Put(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	c, res, err := p.initHandler()
	if err != nil {
		return
	}
	defer WriteResponse(w, res)

	vars := mux.Vars(r)
	id, ok := vars[ID]
	if !ok {
		res.AddError(ErrorNoID)
		res.SetStatusCode(http.StatusBadRequest)
		return
	}

	m, err := c.View(ctx, id)
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

	if a, ok := m.(Authoriser); ok {
		if err := a.Authorise(ctx, Action{Method: http.MethodPut}); err != nil {
			res.AddError(err)
			res.SetStatusCode(http.StatusUnauthorized)
			return
		}
	}

	err = json.NewDecoder(r.Body).Decode(&RestrictedModel{m})
	if err != nil {
		res.AddError(ErrorMalformedJSON)
		res.SetStatusCode(http.StatusBadRequest)
		return
	}

	err = c.Update(ctx, m.PrimaryKey(), m)
	if err != nil {
		res.AddError(fmt.Errorf("failed to update Model: %s", err.Error()))
		res.SetStatusCode(http.StatusInternalServerError)
		return
	}

	res.AddModel(m)
}

// Delete handles deleting the Model specified by the mux var "id"
func (p *Path) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	c, res, err := p.initHandler()
	if err != nil {
		return
	}
	defer WriteResponse(w, res)

	id, ok := mux.Vars(r)[ID]
	if !ok {
		res.AddError(ErrorNoID)
		res.SetStatusCode(http.StatusBadRequest)
		return
	}

	m, err := c.View(ctx, id)
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

	if a, ok := m.(Authoriser); ok {
		if err := a.Authorise(ctx, Action{Method: http.MethodDelete}); err != nil {
			res.AddError(err)
			res.SetStatusCode(http.StatusUnauthorized)
			return
		}
	}

	err = c.Delete(ctx, id)
	if err != nil {
		res.AddError(fmt.Errorf("failed to delete Model: %s", err.Error()))
		res.SetStatusCode(http.StatusInternalServerError)
		return
	}

	res.AddModel(m)
}
