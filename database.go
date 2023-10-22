package rebugpager

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

type Database struct {
	client *firestore.Client
	ctx    context.Context
}

func NewFirestore() (*Database, error) {
	ctx := context.Background()
	sa := option.WithCredentialsFile("./serviceAccount.json")
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
