package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/data"
	"github.com/BunnyTheLifeguard/greenlight/internal/jsonlog"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
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
		user         string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
}

// Application struct to hold dependencies for HTTP handlers, helpers & middleware
type application struct {
	config config
	logger *jsonlog.Logger
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
	flag.StringVar(&cfg.db.user, "db-user", os.Getenv("USER"), "Collection Data")

	// Connection pool cli flags
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "MongoDB max open connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "MongoDB max connection idle time")

	// Rate limiter cli flags
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enabled rate limiter")

	flag.Parse()

	// Initialize a new jsonlog.Logger for messages above INFO severity level
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := openDB(ctx, cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Disconnect(ctx)

	logger.PrintInfo("database connection pool established", nil)

	dataColl := openCollection(db, cfg, cfg.db.data)
	userColl := openCollection(db, cfg, cfg.db.user)

	// Add text indexes for search functionality
	_, err = dataColl.Indexes().CreateOne(context.Background(), mongo.IndexModel{Keys: bson.D{{"title", "text"}, {"genres", "text"}}})
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	// Flag username & email as unique
	_, err = userColl.Indexes().CreateMany(
		context.Background(),
		[]mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "name", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "email", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
	)
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	// Declare an instance of the application struct containing config struct & logger
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(dataColl, userColl),
	}

	err = app.serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
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
