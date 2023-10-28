package rebugpager

import (
	"context"
	"errors"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Database struct {
	client *firestore.Client
	ctx    context.Context
}

type Query struct {
	path  string
	op    string
	value interface{}
}

type OrderBy struct {
	path string
	dir  firestore.Direction
}

func NewFirestore() (*Database, error) {
	ctx := context.Background()
	sa := option.WithCredentialsJSON([]byte(os.Getenv("FIREBASE_AUTH")))
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return nil, err
	}
	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}
	db := Database{
		client: client,
		ctx:    ctx,
	}
	return &db, nil
}

func (db *Database) GetDoc(path string, target any) error {
	if ref := db.client.Doc(path); ref == nil {
		return errors.New("Invalid doc path")
	} else if snap, err := ref.Get(db.ctx); err != nil {
		return err
	} else if err := snap.DataTo(target); err != nil {
		return err
	}
	return nil
}

func (db *Database) QueryDocs(path string, queries []Query, orderBy OrderBy) ([]map[string]any, error) {
	results := []map[string]any{}
	if len(queries) == 0 {
		return results, errors.New("Function meant to be run with queries.")
	}
	ref := db.client.Collection(path)
	if ref == nil {
		return results, errors.New("Invalid collection path")
	}
	query := ref.Where(queries[0].path, queries[0].op, queries[0].value)
	for _, q := range queries[1:] {
		query = query.Where(q.path, q.op, q.value)
	}
	iter := query.OrderBy(orderBy.path, orderBy.dir).Documents(db.ctx)
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return results, err
		}
		results = append(results, snap.Data())
		results[len(results)-1]["docPath"] = strings.Split(snap.Ref.Path, "(default)/documents/")[1]
	}
	return results, nil
}

func (db *Database) WriteDoc(path string, data any) error {
	if ref := db.client.Doc(path); ref == nil {
		return errors.New("Invalid doc path")
	} else if _, err := ref.Set(db.ctx, data); err != nil {
		return err
	}
	return nil
}

func (db *Database) AddDoc(path string, data any) error {
	if ref := db.client.Collection(path); ref == nil {
		return errors.New("Invalid collection path")
	} else if _, _, err := ref.Add(db.ctx, data); err != nil {
		return err
	}
	return nil
}

func (db *Database) DeleteDoc(path string) error {
	if ref := db.client.Doc(path); ref == nil {
		return errors.New("Invalid doc path")
	} else if _, err := ref.Delete(db.ctx); err != nil {
		return err
	}
	return nil
}
