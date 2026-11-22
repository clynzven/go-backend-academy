// Command adduser creates a user who can log in to the API.
//
//	DATABASE_URL=... go run ./cmd/adduser -tenant acme -email alice@acme.test -password 'correct horse'
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/clynzven/go-backend-academy/internal/auth"
	"github.com/clynzven/go-backend-academy/internal/db"
	"github.com/clynzven/go-backend-academy/migrations"
)

func main() {
	tenant := flag.String("tenant", "", "the user's tenant, like acme")
	email := flag.String("email", "", "login email")
	password := flag.String("password", "", "login password")
	flag.Parse()
	if *tenant == "" || *email == "" || *password == "" {
		log.Fatal("usage: adduser -tenant acme -email alice@acme.test -password 'correct horse'")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if _, err := db.Migrate(ctx, pool, migrations.FS); err != nil {
		log.Fatal(err)
	}
	id, err := auth.NewService(pool, nil).AddUser(ctx, *tenant, *email, *password)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("user %d: %s (tenant %s)\n", id, *email, *tenant)
}
