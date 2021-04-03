package crudley

import (
	"encoding/json"
	"net/http"
)

// Response is the container for all output of the REST handlers
type Response struct {
	Results  []Model `json:"results,omitempty"`
	Error   string `json:"error,omitempty"`
	code int
}

// SetStatusCode sets the http status code for the request
func (r *Response) SetStatusCode(code int) {
	r.code = code
}

// GetStatusCode returns the set http status code for the response
func (r *Response) GetStatusCode() int {
	if r.code == 0 {
		return http.StatusOK
	}
	return r.code
}

// AddModel adds models to the Response
func (r *Response) AddModel(models ...Model) {
	r.Results = append(r.Results, models...)
}

// AddError adds errors to the response
func (r *Response) AddError(errors ...error) {
	for _, err := range errors {
		if r.Error != "" {
			r.Error += ", "
		}
		r.Error += err.Error()
	}
}

// ResponseMiddleware handles writing the api response format to the http.ResponseWriter
func WriteResponse(w http.ResponseWriter, res *Response) {
	// output response
	buf, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "could not output response: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	if res.GetStatusCode() != http.StatusOK {
		w.WriteHeader(res.GetStatusCode())
	}
	w.Write(buf)
}
