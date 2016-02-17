package mem

import (
	"encoding/json"
	"reflect"

	"stablelib.com/v1/uuid"

	"github.com/avct/rest"
	"github.com/avct/rest/stores/backend/memdb"
)

// NewStore returns a new memstore instance
func NewStore() rest.Store {
	return &Store{db: memdb.New()}
}

// Store is a memdb implementation of the rest.Store interface
type Store struct {
	db *memdb.Memdb
}

// Collection retrieves or creates a new collection from the Store
func (s *Store) Collection(mdl rest.Model) (rest.Collection, error) {
	return &Collection{
		col:   s.db.Collection(mdl.GetName()),
		model: mdl,
	}, nil
}

// Collection is the rest.Collection memdb implementation
type Collection struct {
	col   *memdb.Collection
	model rest.Model
}

// Update an existing Model in the memdb
func (c *Collection) Update(id string, model rest.Model) error {
	return c.col.Set(id, model)
}

// Create creates a new instance of the Model, and saves it to the Collection
func (c *Collection) Create(crFunc rest.CreaterFunc) error {
	id := c.id()
	mdl, err := crFunc(id)
	if err != nil {
		return err
	}
	err = c.col.Set(id, mdl)
	return err
}

// Delete removes a Model from the collection
func (c *Collection) Delete(id string) error {
	return c.col.Remove(id)
}

// Scan iterates over all items in the collection from memdb
func (c *Collection) Scan(scanner rest.ScannerFunc) error {
	for _, doc := range c.col.AllRaw() {
		m := c.model.New("")
		err := json.Unmarshal(doc, m)
		if err != nil {
			return err
		}
		err = scanner(m)
		if err != nil {
			return err
		}
	}
	return nil
}

// Search accepts a partial model as a query, scans the Collection, passing all
// matched Models to the rest.ScannerFunc. this Store implementation does not
// support secondary indexes
func (c *Collection) Search(partialModel rest.Model, scanner rest.ScannerFunc) (int, error) {
	var found bool
	var count int
	err := c.Scan(rest.ScannerFunc(func(scanModel rest.Model) error {
		count++
		found = true
		sModelValue := reflect.ValueOf(scanModel).Elem()
		pModelValue := reflect.ValueOf(partialModel).Elem()
		var matched = true
		for i := 0; i < sModelValue.NumField(); i++ {
			sModelFieldValue := sModelValue.Field(i)
			pModelFieldValue := pModelValue.Field(i)
			if !reflect.DeepEqual(pModelFieldValue.Interface(), reflect.Zero(pModelFieldValue.Type()).Interface()) {
				if !reflect.DeepEqual(sModelFieldValue.Interface(), pModelFieldValue.Interface()) {
					matched = false
				}
			}
		}
		if matched {
			return scanner(scanModel)
		}
		return nil
	}))
	if found {
		return count, err
	}
	return count, nil
}

// View retrieves a Model from the memdb
func (c *Collection) View(id string) (rest.Model, error) {
	mdl := c.model.New(id)
	found, err := c.col.Doc(id, mdl)
	if !found {
		return nil, nil
	}
	return mdl, err
}

func (c *Collection) Query() rest.Query {
	return &Query{
		col: c,
	}
}

type kv struct {
	k string
	v interface{}
}

// Query allows the user to construct complex queries against a collection
type Query struct {
	col            *Collection
	eq, ne, gt, lt []kv
	has, sort      string
	limit, skip    int
}

func (q *Query) Equal(key string, val interface{}) {
	q.eq = append(q.eq, kv{key, val})
}

func (q *Query) NotEqual(key string, val interface{}) {
	q.ne = append(q.ne, kv{key, val})
}

func (q *Query) GreaterThan(key string, val interface{}) {
	q.gt = append(q.gt, kv{key, val})
}

func (q *Query) LessThan(key string, val interface{}) {
	q.lt = append(q.lt, kv{key, val})
}

func (q *Query) Limit(n int) {
	q.limit = n
}

func (q *Query) Skip(n int) {
	q.skip = n
}

func (q *Query) Has(key string) {
	q.has = key
}

func (q *Query) Sort(by string) {
	q.sort = by
}

// Execute runs the Query
func (q *Query) Execute() ([]rest.Model, error) {
	var out []rest.Model
	var i int
	q.col.Scan(func(m rest.Model) error {
		i++
		if i < q.skip {
			return nil
		}
		if i > q.limit && q.limit != 0 {
			return nil
		}
		mValue := reflect.ValueOf(m).Elem()
		pass := check(mValue, q)
		if pass {
			out = append(out, m)
		}
		return nil
	})
	return out, nil
}

func check(mValue reflect.Value, q *Query) bool {
	mType := mValue.Type()
	var pass bool
	var checks int
	for i := 0; i < mValue.NumField(); i++ {
		tag := mType.Field(i).Tag.Get("json")
		if tag == "" {
			if mValue.Field(i).Kind() == reflect.Struct {
				pass = check(mValue.Field(i), q)
				if !pass {
					return false
				}
			}
		}
		checks = 0
		if tag == q.has && q.has != "" {
			if reflect.DeepEqual(mValue.Field(i).Interface(), reflect.Zero(mValue.Field(i).Type()).Interface()) {
				return false
			}
		}
		for _, kv := range q.eq {
			if kv.k == tag {
				checks++
				if reflect.DeepEqual(kv.v, mValue.Field(i).Interface()) {
					pass = true
				} else {
					pass = false
				}
			}
		}
		if pass == false && checks != 0 {
			return false
		}
		checks = 0
		for _, kv := range q.ne {
			if kv.k == tag {
				checks++
				if !reflect.DeepEqual(kv.v, mValue.Field(i).Interface()) {
					pass = true
				} else {
					pass = false
				}
			}
		}
		if pass == false && checks != 0 {
			return false
		}
		checks = 0
		for _, kv := range q.gt {
			if kv.k == tag {
				checks++
				mKVval := reflect.ValueOf(kv.v)
				switch mValue.Field(i).Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					mFieldIntval := mValue.Field(i).Int()
					switch mKVval.Kind() {
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						mKVIntVal := mKVval.Int()
						if mFieldIntval > mKVIntVal {
							pass = true
						} else {
							pass = false
						}
					default:
						pass = false
					}
				case reflect.Float32, reflect.Float64:
					mFieldFloatval := mValue.Field(i).Float()
					switch mKVval.Kind() {
					case reflect.Float32, reflect.Float64:
						mKVFloatVal := mKVval.Float()
						if mFieldFloatval > mKVFloatVal {
							pass = true
						} else {
							pass = false
						}
					default:
						pass = false
					}
				default:
					pass = false
				}
			}
		}
		if pass == false && checks != 0 {
			return false
		}
		checks = 0
		for _, kv := range q.lt {
			if kv.k == tag {
				checks++
				mKVval := reflect.ValueOf(kv.v)
				switch mValue.Field(i).Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					mFieldIntval := mValue.Field(i).Int()
					switch mKVval.Kind() {
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						mKVIntVal := mKVval.Int()
						if mFieldIntval < mKVIntVal {
							pass = true
						} else {
							pass = false
						}
					default:
						pass = false
					}
				case reflect.Float32, reflect.Float64:
					mFieldFloatval := mValue.Field(i).Float()
					switch mKVval.Kind() {
					case reflect.Float32, reflect.Float64:
						mKVFloatVal := mKVval.Float()
						if mFieldFloatval < mKVFloatVal {
							pass = true
						} else {
							pass = false
						}
					default:
						pass = false
					}
				}
			}
		}
		if pass == false && checks != 0 {
			return false
		}
		pass = false
	}
	return true
}

func (c *Collection) id() string {
	return uuid.New()
}
