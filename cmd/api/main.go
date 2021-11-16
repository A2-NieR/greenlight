package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/data"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Application version number will be generated automatically at build time later
const version = "1.0.0"

// Config struct to hold all configurations settings of the application, will be read from cli-flags
type config struct {
	port int
	env  string
	db   struct {
		uri          string
		maxOpenConns int
		maxIdleTime  string
		name         string
		data         string
	}
}

// Application struct to hold dependencies for HTTP handlers, helpers & middleware
type application struct {
	config config
	logger *log.Logger
	models data.Models
}

func init() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	var cfg config

	// Read value of port & env cli-flags into config-struct. Default to port 4000 and environment "development" if no flags provided
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.uri, "db-uri", os.Getenv("MONGODB_URI"), "MongoDB URI")
	flag.StringVar(&cfg.db.name, "db-name", os.Getenv("DB"), "DB Name")
	flag.StringVar(&cfg.db.data, "db-data", os.Getenv("DATA"), "Collection Data")

	// Connection pool cli flags
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "MongoDB max open connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "MongoDB max connection idle time")

	flag.Parse()

	// Initialize a new logger that writes messages to the standard out stream, prefixed with current date & time
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := openDB(ctx, cfg)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Disconnect(ctx)

	logger.Printf("database connection pool established")

	dataColl := openCollection(db, cfg, cfg.db.data)

	// Declare an instance of the application struct containing config struct & logger
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(dataColl),
	}

	// Declare a HTTP server with timeout settings that listens on provided port in config struct and uses servemux from above as handler
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start the HTTP server
	logger.Printf("Starting %s server on %s", cfg.env, srv.Addr)
	err = srv.ListenAndServe()
	logger.Fatal(err)
}

func openDB(ctx context.Context, cfg config) (*mongo.Client, error) {
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}

	client, err := mongo.NewClient(options.Client().SetMaxPoolSize(uint64(cfg.db.maxOpenConns)).SetMaxConnIdleTime(duration).ApplyURI(cfg.db.uri))

	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	return client, nil
}

func openCollection(client *mongo.Client, cfg config, coll string) *mongo.Collection {
	collection := client.Database(cfg.db.name).Collection(coll)
	return collection
}
