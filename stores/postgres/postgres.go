package postgres

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/arussellsaw/crudley"
)

var _ crudley.Store = &Store{}

func NewStore(host, port, db, user, pass string) (*Store, error) {
	// connection string
	psqlconn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s", host, port, user, pass, db)

	// open database
	c, err := sql.Open("postgres", psqlconn)
	if err != nil {
		return nil, err
	}

	// check db
	err = c.Ping()
	if err != nil {
		return nil, err
	}

	return &Store{}, nil
}

type Store struct {
	conn *sql.DB
}

func (s *Store) Collection(m crudley.Model) (crudley.Collection, error) {
	return &Collection{
		tableName: m.GetName(),
		c:         s.conn,
		m:         m,
	}, nil
}

type Collection struct {
	tableName string
	c         *sql.DB
	m         crudley.Model
	crudley.Collection
}

func (c *Collection) View(ctx context.Context, id string) (crudley.Model, error) {
	m := c.m.New(id)
	c.c.QueryRowContext(ctx, "SELECT * FROM @table WHERE id = @id").Scan(m)
	return nil, nil
}
