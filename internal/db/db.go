package db

import (
	"database/sql"
	"errors"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
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

		dbURL := url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(user, password),
			Host:     net.JoinHostPort(host, strconv.Itoa(port)),
			Path:     "/postgres",
			RawQuery: "sslmode=disable",
		}

		log.Printf("Connecting to Postgres at %s%s", dbURL.Host, dbURL.Path)

		newdb, err := sql.Open("postgres", dbURL.String())
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
