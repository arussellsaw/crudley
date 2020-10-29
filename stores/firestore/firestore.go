package firestore

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"github.com/arussellsaw/crudley"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"strings"
	"time"
)

var _ crudley.Store = &Store{}

func NewStore(ctx context.Context, projectID string) (crudley.Store, error) {
	var err error
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &Store{
		c:   fs,
		ctx: ctx,
	}, nil
}

type Store struct {
	c *firestore.Client
	// TODO: pass this per method
	ctx context.Context
}

func (s *Store) Collection(m crudley.Model) (crudley.Collection, error) {
	return &Collection{
		col:   s.c.Collection(m.GetName()),
		ctx:   s.ctx,
		Model: m,
	}, nil
}

type Collection struct {
	col   *firestore.CollectionRef
	Model crudley.Model
	ctx   context.Context
}

func (c *Collection) View(id string) (crudley.Model, error) {
	ds, err := c.col.Doc(id).Get(c.ctx)
	if err != nil {
		return nil, err
	}
	m := c.Model.New("")
	err = ds.DataTo(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (c *Collection) Update(id string, m crudley.Model) error {
	// TODO: right now this overwrites all fields so updates must contain
	// the whole model to not blank fields. i think i can iterate over non-empty
	// struct fields using fatih/structs to only update non-zero fields, but i'll do that later
	_, err := c.col.Doc(id).Set(c.ctx, m)
	return err
}

func (c *Collection) Delete(id string) error {
	_, err := c.col.Doc(id).Delete(c.ctx)
	return err
}

func (c *Collection) Scan(fn crudley.ScannerFunc) error {
	iter := c.col.Documents(c.ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		m := c.Model.New("")
		err = doc.DataTo(m)
		if err != nil {
			return err
		}
		err = fn(m)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Collection) Create(fn crudley.CreaterFunc) error {
	id := uuid.New().String()
	m, err := fn(id)
	if err != nil {
		return err
	}
	_, err = c.col.Doc(id).Set(c.ctx, m)
	return err
}

func (c *Collection) Search(m crudley.Model, fn crudley.ScannerFunc) (int, error) {
	return 0, errors.New("not implemented")
}

func (c *Collection) Query() crudley.Query {
	return &Query{
		col:   c.col,
		ctx:   c.ctx,
		Model: c.Model,
		q:     c.col.Query,
	}
}

type Query struct {
	col   *firestore.CollectionRef
	Model crudley.Model
	ctx   context.Context
	q     firestore.Query
}

func (q *Query) Equal(key string, val interface{}) {
	q.q = q.q.Where(key, "==", val)
}

func (q *Query) NotEqual(key string, val interface{}) {
	q.q = q.q.Where(key, "!=", val)
}

func (q *Query) GreaterThan(key string, val interface{}) {
	q.q = q.q.Where(key, ">", val)
}

func (q *Query) LessThan(key string, val interface{}) {
	q.q = q.q.Where(key, "<", val)
}

func (q *Query) Limit(n int) {
	q.q = q.q.Limit(n)
}

func (q *Query) Skip(n int) {
	q.q = q.q.Offset(n)
}

func (q *Query) Has(key string) {
	// this will only match on values where the field exists, so should satisfy the API
	q.q = q.q.Where(key, "not-in", "i really hope this value doesnt exist") // yikes
}

func (q *Query) Sort(by string) {
	field := strings.TrimPrefix(by, "-")
	sort := firestore.Asc
	if strings.HasPrefix(by, "-") {
		sort = firestore.Desc
	}
	q.q = q.q.OrderBy(field, sort)
}

func (q *Query) Execute() ([]crudley.Model, error) {
	out := []crudley.Model{}
	iter := q.q.Documents(q.ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		m := q.Model.New("")
		err = doc.DataTo(m)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func NewBase(id string) Base {
	return Base{ID: id, Created: time.Now()}
}

type Base struct {
	ID      string    `firestore:"id,omitempty" json:"id,omitempty"`
	Created time.Time `firestore:"createdt,omitempty" json:"created,omitempty"`
	Deleted time.Time `firestore:"deletedt,omitempty" json:"deleted,omitempty"`
}

// New creates a new Base model, you should override this on your parent struct, but call it
// when initialising the embedded Base `return MyCoolModel{Base:c.Base.New(id)}`
func (b *Base) New(id string) crudley.Model {
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
	b.Deleted = time.Now()
}

func (b *Base) IsDeleted() bool {
	return !b.Deleted.IsZero()
}
