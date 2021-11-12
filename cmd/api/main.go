package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Application version number will be generated automatically at build time later
const version = "1.0.0"

// Config struct to hold all configurations settings of the application, will be read from cli-flags
type config struct {
	port int
	env  string
}

// Application struct to hold dependencies for HTTP handlers, helpers & middleware
type application struct {
	config config
	logger *log.Logger
}

func main() {
	var cfg config

	// Read value of port & env cli-flags into config-struct. Default to port 4000 and environment "development" if no flags provided
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.Parse()

	// Initialize a new logger that writes messages to the standard out stream, prefixed with current date & time
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	// Declare an instance of the application struct containing config struct & logger
	app := &application{
		config: cfg,
		logger: logger,
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
	err := srv.ListenAndServe()
	logger.Fatal(err)
}
