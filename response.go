package crudley

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gorilla/context"
)

// Response is the container for all output of the REST handlers
type Response struct {
	ID       string
	Models   []Model
	Errors   []string
	Code     int
	Total    int
	Message  string
	Error    string
	Duration time.Duration
}

// SetStatusCode sets the http status code for the request
func (r *Response) SetStatusCode(code int) {
	r.Code = code
}

// GetStatusCode returns the set http status code for the response
func (r *Response) GetStatusCode() int {
	if r.Code == 0 {
		return http.StatusOK
	}
	return r.Code
}

// AddModel adds models to the Response
func (r *Response) AddModel(models ...Model) {
	r.Models = append(r.Models, models...)
}

// AddError adds errors to the response
func (r *Response) AddError(errors ...error) {
	for _, err := range errors {
		r.Errors = append(r.Errors, err.Error())
		// set main message to last error sent
		r.Error = err.Error()
	}
}

// ResponseMiddleware handles writing the api response format to the http.ResponseWriter
func (p *Path) ResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// initialize response
		id := uuid.New()
		context.Set(r, "requestid", id)
		res := &Response{ID: id.String(), Code: http.StatusOK}
		context.Set(r, "response", res)
		// perform api request
		start := time.Now()
		next.ServeHTTP(w, r)
		res.Duration = time.Since(start)
		// output response
		var buf []byte
		var err error
		if p.MarshalResponse != nil {
			buf, err = p.MarshalResponse(res)
		} else {
			buf, err = json.Marshal(res)
		}
		if err != nil {
			http.Error(w, "could not output response: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		if res.GetStatusCode() != http.StatusOK {
			w.WriteHeader(res.GetStatusCode())
		}
		w.Write(buf)
	})
}
