package crudley

import (
	"net/http"
	"time"
)

/*
	Rest is a toolkit for building restful APIs focused on dealing with application
	and business logic whilst avoiding dealing with messy and repettetive http/REST
	handling.

	The Store/Collection interfaces are aimed at providing swappable storage backends
	with no code changes on actual models and application logic (though some struct
	tags would have to be modified depending on eventual serialization format).
*/

// Model is the minimal interface your API's documents must satisfy to allow the
// RESTful handlers to manage modification and storage. there are further optional
// interfaces you can satisfy [PreSaver, PostSaver, Validator, Permissioner] to
// allow more complex custom behaviours to your documents
type Model interface {
	// New returns a new instance of the Model with the specified ID and necessary
	// fields initialized
	New(id string) Model
	// GetName retrieves the name of the Model
	GetName() string
	// PrimaryKey returns the Primary Key for a model
	PrimaryKey() string
	// Delete modifies the document to mark deletion, as actual deletion is not supported
	Delete()
	// IsDeleted determines whether a Model has been marked as Deleted
	IsDeleted() bool
}

// Collection represents a set of Models from a Store. this handles Model creation
// and Modification. it also usually contains a connection or collection instance
// from things like mongodb, so may contain state. this should be checked before usage
type Collection interface {
	// View retrieves a model from the Collection and deserializes it into the
	// provided Model instance
	View(id string) (Model, error)
	// Update an existing model in the collection
	Update(id string, m Model) error
	// Delete removes a model from the collection
	Delete(id string) error
	// Scan iterates through all Models in the collection, running the provided
	// ScannerFunc on each serialized Model
	Scan(ScannerFunc) error
	// Create handles creating a new model
	Create(CreaterFunc) error
	// Search accepts a partial Model and a ScannerFunc to query on alternate indexes
	Search(Model, ScannerFunc) (int, error)
	// Query retrieves an object that can be used to perform advanced queries on the Store
	Query() Query
}

// Query represents a way to build advanced queries on a Collection, each method adding a predicate to the query
type Query interface {
	Equal(key string, val interface{})
	NotEqual(key string, val interface{})
	GreaterThan(key string, val interface{})
	LessThan(key string, val interface{})
	Limit(int)
	Skip(int)
	Sort(string)
	Has(string)
	Execute() ([]Model, error)
}

// Store represents a storage service for Models, this generally does not need to
// contain state or an active connection, as it is usually just used to store info
// used to retrieve a Collection.
type Store interface {
	// Collection retrieves the specified Collection from the Store, using the Model
	// to unmarshal into.
	Collection(m Model) (Collection, error)
}

// Permissioner is an optional interface that a Model can also implement, this
// allows the Model to be restricted by custom authentication and scopes. the
// recommended implementation is reading permissions from a value stored in the
// request mux.context, but could also use values in the request to look up
// permission levels from an external service.
type Permissioner interface {
	Permissions(*http.Request) bool
}

// Validator is an optional interface that a Model can implement. Validator is
// used to check validity and safety of fields on the Model, such as checking
// email fields are valid, or checking references exist.
type Validator interface {
	Validate(Store) error
}

// PreSaver is an optional interface that a Model can implement. PreSaver.Presave is
// called before persisting a modified Model to the Collection. This method can be
// used to broadcast updates via message queues, perform logging, update references etc.
type PreSaver interface {
	PreSave(Store) error
}

// PostSaver is an optional interface that a Model can implement. PostSaver.Postsave is
// called after persisting a modified Model to the Collection. This method can be
// used to broadcast updates via message queues, perform logging, update references etc.
type PostSaver interface {
	PostSave(Store) error
}

// ScannerFunc is used to iterate over Models for queries. they are used in the
// multiple Model response handlers. depending on the Store implementation Query
// may use a ScannerFunc to filter an entire collection of Models, if the database
// itself is unable to perform an operation like that. (see memdb vs mongodb Stores)
type ScannerFunc func(Model) error

// CreaterFunc is used by Collection.Create() to initialize a new Model with appropriate
// defaults and validations set
type CreaterFunc func(id string) (Model, error)

// HandlerFunc is an extended http.Handler to allow passed handlers to use the Path's properties
type HandlerFunc func(handler *Path, w http.ResponseWriter, r *http.Request)

// NotFoundError is returned when a store cannot lcoate a document
type NotFoundError string

func (e NotFoundError) Error() string {
	return string(e)
}

func NewBase(id string) Base {
	return Base{ID: id, CreatedAt: time.Now()}
}

type Base struct {
	ID        string    `bson:"_id,omitempty" json:"id,omitempty"`
	CreatedAt time.Time `bson:"created_at,omitempty" json:"created_at,omitempty"`
	Deleted   bool      `bson:"deleted,omitempty" json:"deleted,omitempty"`
	DeletedAt time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// New creates a new Base model, you should override this on your parent struct, but call it
// when initialising the embedded Base `return MyCoolModel{Base:c.Base.New(id)}`
func (b *Base) New(id string) Model {
	nb := NewBase(id)
	return &nb
}

// GetName should be overridden to return the name of your collection
func (b *Base) GetName() string {
	panic("must override Name() on Base model")
	return "base"
}

func (b *Base) PrimaryKey() string {
	return b.ID
}

func (b *Base) Delete() {
	b.Deleted = true
	b.DeletedAt = time.Now()
}

func (b *Base) IsDeleted() bool {
	return b.Deleted
}
