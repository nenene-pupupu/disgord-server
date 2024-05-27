package controller

import (
	"context"
	"log"

	"disgord/ent"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
)

var (
	client *ent.Client
	ctx    context.Context
)

type Controller struct{}

func New() *Controller {
	var err error

	client, err = ent.Open(dialect.SQLite, "file:disgord.db?cache=shared&_fk=1")
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}

	ctx = context.Background()

	// Run the automatic migration tool to create all schema resources.
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	return &Controller{}
}

func (*Controller) Close() error {
	return client.Close()
}
