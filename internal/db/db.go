package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/dota2classic/d2c-go-models/util"
	_ "github.com/lib/pq"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

var (
	db         *sql.DB
	clientInit sync.Once
)

func ConnectAndMigrate() *sql.DB {
	clientInit.Do(func() {
		host := os.Getenv("POSTGRES_HOST")
		port := util.GetEnvInt("POSTGRES_PORT", 5432)
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")

		dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", user, password, host, port, "postgres")

		log.Println(dbURL)

		newdb, err := sql.Open("postgres", dbURL)
		if err != nil {
			log.Fatalf("failed to connect to db: %v", err)
		}

		if err := newdb.Ping(); err != nil {
			log.Fatalf("failed to ping db: %v", err)
		}

		runMigrations(newdb)

		db = newdb

	})
	return db
}

func runMigrations(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("failed to create migrate driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://db/migrations",
		"postgres", driver)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	log.Println("Migrations applied successfully")
}
