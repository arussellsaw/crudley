package mongo

import (
	"fmt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/arussellsaw/crudley"
)

// NewStore creates a new mongodb backed rest.Store
func NewStore(hosts, database, user, pass string) (rest.Store, error) {
	sess, err := mgo.Dial(hosts)
	if err != nil {
		return nil, err
	}
	store := &Store{
		session:  sess,
		user:     user,
		pass:     pass,
		database: database,
	}
	db := store.session.DB(store.database)
	if store.user != "" && store.pass != "" {
		err = db.Login(store.user, store.pass)
		if err != nil {
			return nil, err
		}
	}
	store.db = db
	return store, nil
}

// Store is a mongodb backed implementation of the rest.Store interface
type Store struct {
	session              *mgo.Session
	db                   *mgo.Database
	user, pass, database string
}

// Collection returns the rest.Collection from mongodb specified by the name and
// rest.Model
func (s *Store) Collection(m rest.Model) (rest.Collection, error) {
	col := s.db.C(m.GetName())
	return &Collection{
		col:   col,
		Model: m,
	}, nil
}

// Collection represents a rest.Collection from mongodb
type Collection struct {
	col   *mgo.Collection
	Model rest.Model
}

// View retrieves a single rest.Model from the Collection
func (c *Collection) View(id string) (rest.Model, error) {
	if id == "" {
		return nil, fmt.Errorf("you must specify a Model id")
	}
	m := c.Model.New("")
	query := c.col.Find(idmap(id))
	err := query.One(m)
	if err == mgo.ErrNotFound {
		return nil, nil
	}
	if m.IsDeleted() {
		return nil, nil
	}
	return m, err
}

func idmap(id string) map[string]string {
	return map[string]string{"_id": id}
}

// Update an existing rest.Model in the Collection
func (c *Collection) Update(id string, m rest.Model) error {
	if id == "" {
		return fmt.Errorf("you must specify a model id")
	}
	err := c.col.Update(idmap(id), m)
	if err == mgo.ErrNotFound {
		return fmt.Errorf("Model not found")
	}
	return err
}

// Delete a rest.Model from the Collection
func (c *Collection) Delete(id string) error {
	if id == "" {
		return fmt.Errorf("you must specify a model id")
	}
	err := c.col.Remove(idmap(id))
	if err == mgo.ErrNotFound {
		return fmt.Errorf("Model not found")
	}
	return err
}

// Scan accepts a function to run on the serialized version of every rest.Model
// in the collection
func (c *Collection) Scan(scanFn rest.ScannerFunc) error {
	query := c.col.Find(bson.M{})
	iter := query.Iter()
	m := c.Model.New("")
	for iter.Next(m) {
		err := scanFn(m)
		if err != nil {
			return err
		}
		m = c.Model.New("")
	}
	return nil
}

// Search accepts a partial rest.Model as a query parameter and applies the passed
// rest.ScannerFunc to the resulting set. This implementation supports secondary
// indexes, so make sure the partial fields are indexed by the rest.Model.Index()
func (c *Collection) Search(partial rest.Model, scanner rest.ScannerFunc) (int, error) {
	query := c.col.Find(partial)
	total, err := query.Count()
	if err != nil {
		return total, err
	}
	iter := query.Iter()
	m := c.Model.New("")
	for iter.Next(m) {
		err := scanner(m)
		m = c.Model.New("")
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// Create accepts a creation function to add a new rest.Model to the collection
func (c *Collection) Create(createFn rest.CreaterFunc) error {
	id := c.id()
	m, err := createFn(id)
	if err != nil {
		return err
	}
	return c.col.Insert(m)
}

// Query returns a rest.Query for building more complex queries against the Collection
func (c *Collection) Query() rest.Query {
	return &Query{
		m:     bson.M{},
		col:   c.col,
		model: c.Model,
	}
}

// Query allows the user to construct complex queries against a collection using
// the provided methods to build an underlying bson.M to pass to the collection.Find()
type Query struct {
	m           bson.M
	model       rest.Model
	skip, limit int
	sort        string
	col         *mgo.Collection
}

// Equal adds a simple key: value to the query map
func (q *Query) Equal(key string, val interface{}) {
	if m, ok := q.m[key].(bson.M); ok {
		if v, ok := m["$in"].([]interface{}); ok {
			m["$in"] = append(v, val)
		} else {
			m["$in"] = []interface{}{val}
		}
	} else {
		q.m[key] = bson.M{"$in": []interface{}{val}}
	}
}

// NotEqual adds a key: {$ne: val} to the query map
func (q *Query) NotEqual(key string, val interface{}) {
	if m, ok := q.m[key].(bson.M); ok {
		if v, ok := m["$nin"].([]interface{}); ok {
			m["$nin"] = append(v, val)
		} else {
			m["$nin"] = []interface{}{val}
		}
	} else {
		q.m[key] = bson.M{"$nin": []interface{}{val}}
	}
}

// GreaterThan adds a key: {$gt: val} to the query map
func (q *Query) GreaterThan(key string, val interface{}) {
	if m, ok := q.m[key].(bson.M); ok {
		m["$gt"] = val
	} else {
		q.m[key] = bson.M{"$gt": val}
	}
}

// LessThan adds a key: {$lt: val} to the query map
func (q *Query) LessThan(key string, val interface{}) {
	if m, ok := q.m[key].(bson.M); ok {
		m["$lt"] = val
	} else {
		q.m[key] = bson.M{"$lt": val}
	}
}

// Has determines that a parameter exists and is not null
func (q *Query) Has(key string) {
	q.m[key] = bson.M{"$exists": true}
}

// Limit sets the maximum number of Models to retrieve from the query
func (q *Query) Limit(n int) {
	q.limit = n
}

// Skip defines the number of Models to skip before adding to the result set
func (q *Query) Skip(n int) {
	q.skip = n
}

// Sort defines the field by which the result set should be sorted
func (q *Query) Sort(by string) {
	q.sort = by
}

// Execute runs the Query
func (q *Query) Execute() ([]rest.Model, error) {
	mdls := []rest.Model{}
	query := q.col.Find(q.m)
	if q.limit != 0 {
		query = query.Limit(q.limit)
	}
	query = query.Skip(q.skip)
	if q.sort != "" {
		query = query.Sort(q.sort)
	}
	iter := query.Iter()
	mdl := q.model.New("")
	for iter.Next(mdl) {
		err := iter.Err()
		if err != nil {
			return mdls, err
		}
		mdls = append(mdls, mdl)
		mdl = q.model.New("")
	}
	return mdls, nil
}

func (c *Collection) id() string {
	return bson.NewObjectId().Hex()
}
