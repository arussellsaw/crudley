package firestore

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"github.com/arussellsaw/crudley"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

var _ rest.Store = &Store{}

func NewStore(ctx context.Context, projectID string) (rest.Store, error) {
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

func (s *Store) Collection(m rest.Model) (rest.Collection, error) {
	return &Collection{
		col:   s.c.Collection(m.GetName()),
		ctx:   s.ctx,
		Model: m,
	}, nil
}

type Collection struct {
	col   *firestore.CollectionRef
	Model rest.Model
	ctx   context.Context
}

func (c *Collection) View(id string) (rest.Model, error) {
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

func (c *Collection) Update(id string, m rest.Model) error {
	_, err := c.col.Doc(id).Set(c.ctx, m)
	return err
}

func (c *Collection) Delete(id string) error {
	_, err := c.col.Doc(id).Delete(c.ctx)
	return err
}

func (c *Collection) Scan(fn rest.ScannerFunc) error {
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

func (c *Collection) Create(fn rest.CreaterFunc) error {
	id := uuid.New().String()
	m, err := fn(id)
	if err != nil {
		return err
	}
	_, err = c.col.Doc(id).Set(ctx, m)
	return err
}

func (c *Collection) Search(m rest.Model, fn rest.ScannerFunc) (int, error) {
	return 0, errors.New("not implemented")
}

func (c *Collection) Query() rest.Query {
	return nil
}
