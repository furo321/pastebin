package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/schema"
	"github.com/joho/godotenv"
	"github.com/unrolled/render"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB
var t *render.Render
var decoder = schema.NewDecoder()

var ticker = time.NewTicker(1 * time.Minute)
var quit = make(chan struct{})

func periodicDelete(postgresql bool) {
	for {
		select {
		case <-ticker.C:
			deleteOldPastes(postgresql)
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func main() {
	var err error

	// load env
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// database
	uri := os.Getenv("DATABASE_URI")
	driver := "sqlite3"

	if strings.HasPrefix(uri, "postgres") {
		driver = "postgres"
	}

	db, err = sql.Open(driver, uri)
	if err != nil {
		log.Fatal(err)
	}

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		port = 8080
	}

	// create background task for paste deletions
	go periodicDelete(driver == "postgres")

	createPasteTable(driver == "postgres")
	startHttp(port)
}
