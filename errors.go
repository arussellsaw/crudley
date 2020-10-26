package crudley

import (
	"errors"
)

var (
	ErrorModelNotFound    = errors.New("Model not found")
	ErrorMalformedJSON    = errors.New("Malformed JSON")
	ErrorValidationFailed = errors.New("Validation failed")
	ErrorPostSaveFailed   = errors.New("PostSave failed")
	ErrorPreSaveFailed    = errors.New("PreSave failed")
	ErrorNoID             = errors.New("ID parameter missing")
	ErrorForbidden        = errors.New("Permission Denied")
)
