// services/vehicle/cmd/migrate/main.go
package main

import (
	"log"
	"os"

	mysqlCfg "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/adammwaniki/bebabeba/services/vehicle/internal/store"
)

func main() {
	// Set up the DB config from environment variables
	cfg := mysqlCfg.Config{
		User:                 os.Getenv("DB_USER"),
		Passwd:               os.Getenv("DB_PASSWORD"),
		Addr:                 os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT"),
		DBName:               os.Getenv("DB_NAME"),
		Net:                  "tcp",
		AllowNativePasswords: true,
		MultiStatements:	true,
		ParseTime:            true,
	}

	// Create a raw DB connection for migrations
	db, err := store.NewRawDB(cfg)
	if err != nil {
		log.Fatal("failed to connect to db: ", err)
	}

	// Create migration-compatible database instance
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		log.Fatal("failed to get db instance: ", err)
	}

	// Initialize migration tool
	m, err := migrate.NewWithDatabaseInstance(
		"file://cmd/migrate/migrations",
		"mysql",
		driver,
	)
	if err != nil {
		log.Fatal("failed to create migration instance: ", err)
	}

	// Handle migration commands
	cmd := os.Args[len(os.Args)-1]
	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		log.Println("Migration up completed successfully")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		log.Println("Migration down completed successfully")
	default:
		log.Fatalf("unknown command: %s (expected 'up' or 'down')", cmd)
	}
}