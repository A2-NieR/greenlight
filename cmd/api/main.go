package main

import (
	"context"
	"expvar"
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/BunnyTheLifeguard/greenlight/internal/data"
	"github.com/BunnyTheLifeguard/greenlight/internal/jsonlog"
	"github.com/BunnyTheLifeguard/greenlight/internal/mailer"
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
		token        string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
}

// Application struct to hold dependencies for HTTP handlers, helpers & middleware
type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func init() {
	err := godotenv.Load(filepath.Join(".env"))
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
	flag.StringVar(&cfg.db.user, "db-user", os.Getenv("USER"), "Collection User")
	flag.StringVar(&cfg.db.token, "db-token", os.Getenv("TOKEN"), "Collection Token")

	// Connection pool cli flags
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "MongoDB max open connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "MongoDB max connection idle time")

	// Rate limiter cli flags
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enabled rate limiter")

	// SMTP config settings
	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("SMTPUSER"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("SMTPPASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <36411819+BunnyTheLifeguard@users.noreply.github.com>", "SMTP sender")

	// CORS flags
	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

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
	tokenColl := openCollection(db, cfg, cfg.db.token)

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
				Keys:    bson.M{"name": 1},
				Options: options.Index().SetUnique(true).SetCollation(&options.Collation{Locale: "en", Strength: 2}),
			},
			{
				Keys:    bson.M{"email": 1},
				Options: options.Index().SetUnique(true).SetCollation(&options.Collation{Locale: "en", Strength: 2}),
			},
		},
	)
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	// Add TimeToLive for expiry field to autodelete expired tokens
	_, err = tokenColl.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "expiry", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(1),
	})
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	// Metrics
	expvar.NewString("version").Set(version)
	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))

	expvar.Publish("database", expvar.Func(func() interface{} {
		type Metrics struct {
			Connections bson.M `json:"connections"`
			Metrics     bson.M `json:"metrics"`
		}
		var result *Metrics

		cmd := bson.D{
			{Key: "serverStatus", Value: 1},
			{Key: "repl", Value: 0},
			{Key: "metrics", Value: 1},
			{Key: "locks", Value: 0},
		}
		opts := options.RunCmd().SetReadPreference(readpref.Primary())

		err := db.Database(cfg.db.name).RunCommand(context.TODO(), cmd, opts).Decode(&result)
		if err != nil {
			logger.PrintFatal(err, nil)
		}

		return result
	}))
	expvar.Publish("timestamp", expvar.Func(func() interface{} {
		return time.Now().Unix()
	}))

	// Declare an instance of the application struct containing config struct & logger
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(dataColl, userColl, tokenColl),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
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

	clientOpts := options.Client().ApplyURI(cfg.db.uri).SetMaxPoolSize(uint64(cfg.db.maxOpenConns)).SetMaxConnIdleTime(duration)

	client, err := mongo.NewClient(clientOpts)

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
