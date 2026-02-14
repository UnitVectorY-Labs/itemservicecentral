package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/database"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/handler"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/middleware"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/schema"
)

// Version is the application version, injected at build time via ldflags
var Version = "dev"

func main() {
	// Set the build version from the build info if not set by the build system
	if Version == "dev" || Version == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
				Version = bi.Main.Version
			}
		}
	}

	if len(os.Args) < 2 {
		// Default to api command
		os.Args = append(os.Args, "api")
	}

	switch os.Args[1] {
	case "api":
		runAPI()
	case "validate":
		runValidate()
	case "migrate":
		runMigrate()
	case "version":
		fmt.Println(Version)
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [flags]\n\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  api       Start the API server")
	fmt.Fprintln(os.Stderr, "  validate  Validate the configuration file")
	fmt.Fprintln(os.Stderr, "  migrate   Run database migrations")
	fmt.Fprintln(os.Stderr, "  version   Print version")
	os.Exit(1)
}

// envOrDefault returns the environment variable value if set and the flag is at its default,
// otherwise returns the flag value.
func envOrDefault(flagVal, flagDefault, envVar string) string {
	if envVal := os.Getenv(envVar); envVal != "" && flagVal == flagDefault {
		return envVal
	}
	return flagVal
}

func runAPI() {
	fs := flag.NewFlagSet("api", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "Path to config file")
	port := fs.String("port", "", "Server port")
	dbHost := fs.String("db-host", "localhost", "Database host")
	dbPort := fs.String("db-port", "5432", "Database port")
	dbName := fs.String("db-name", "", "Database name")
	dbUser := fs.String("db-user", "", "Database username")
	dbPassword := fs.String("db-password", "", "Database password")
	dbSSLMode := fs.String("db-sslmode", "disable", "SSL mode")
	fs.Parse(os.Args[2:])

	*configPath = envOrDefault(*configPath, "config.yaml", "ISC_CONFIG")
	*port = envOrDefault(*port, "", "ISC_PORT")
	*dbHost = envOrDefault(*dbHost, "localhost", "ISC_DB_HOST")
	*dbPort = envOrDefault(*dbPort, "5432", "ISC_DB_PORT")
	*dbName = envOrDefault(*dbName, "", "ISC_DB_NAME")
	*dbUser = envOrDefault(*dbUser, "", "ISC_DB_USER")
	*dbPassword = envOrDefault(*dbPassword, "", "ISC_DB_PASSWORD")
	*dbSSLMode = envOrDefault(*dbSSLMode, "disable", "ISC_DB_SSLMODE")

	if *dbName == "" {
		log.Fatal("database name is required: set -db-name or ISC_DB_NAME")
	}
	if *dbUser == "" {
		log.Fatal("database user is required: set -db-user or ISC_DB_USER")
	}
	if *dbPassword == "" {
		log.Fatal("database password is required: set -db-password or ISC_DB_PASSWORD")
	}

	dbPortInt, err := strconv.Atoi(*dbPort)
	if err != nil {
		log.Fatalf("invalid db-port: %v", err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := config.Validate(cfg); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	// Override port from flag/env if set
	if *port != "" {
		p, err := strconv.Atoi(*port)
		if err != nil {
			log.Fatalf("invalid port: %v", err)
		}
		cfg.Server.Port = p
	}

	db, err := database.Connect(*dbHost, dbPortInt, *dbName, *dbUser, *dbPassword, *dbSSLMode)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, cfg.Tables); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	store := database.NewStore(db)

	h, err := handler.New(store, cfg.Tables)
	if err != nil {
		log.Fatalf("failed to create handler: %v", err)
	}

	jwtMw, err := middleware.NewJWTMiddleware(
		cfg.Server.JWT.Enabled,
		cfg.Server.JWT.JWKSUrl,
		cfg.Server.JWT.Issuer,
		cfg.Server.JWT.Audience,
	)
	if err != nil {
		log.Fatalf("failed to create JWT middleware: %v", err)
	}

	mux := http.NewServeMux()
	h.SetupRoutes(mux)

	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.Server.Port),
		Handler: jwtMw.Handler(mux),
	}

	log.Printf("itemservicecentral %s starting on port %d", Version, cfg.Server.Port)
	log.Printf("loaded %d table(s)", len(cfg.Tables))

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	log.Println("server stopped")
}

func runValidate() {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "Path to config file")
	fs.Parse(os.Args[2:])

	*configPath = envOrDefault(*configPath, "config.yaml", "ISC_CONFIG")

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := config.Validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		os.Exit(1)
	}

	// Compile all JSON schemas to verify they are valid
	for _, t := range cfg.Tables {
		if _, err := schema.Compile(t.Schema); err != nil {
			fmt.Fprintf(os.Stderr, "table %q: schema error: %v\n", t.Name, err)
			os.Exit(1)
		}
	}

	fmt.Println("Configuration is valid")
}

func runMigrate() {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "Path to config file")
	dbHost := fs.String("db-host", "localhost", "Database host")
	dbPort := fs.String("db-port", "5432", "Database port")
	dbName := fs.String("db-name", "", "Database name")
	dbUser := fs.String("db-user", "", "Database username")
	dbPassword := fs.String("db-password", "", "Database password")
	dbSSLMode := fs.String("db-sslmode", "disable", "SSL mode")
	fs.Parse(os.Args[2:])

	*configPath = envOrDefault(*configPath, "config.yaml", "ISC_CONFIG")
	*dbHost = envOrDefault(*dbHost, "localhost", "ISC_DB_HOST")
	*dbPort = envOrDefault(*dbPort, "5432", "ISC_DB_PORT")
	*dbName = envOrDefault(*dbName, "", "ISC_DB_NAME")
	*dbUser = envOrDefault(*dbUser, "", "ISC_DB_USER")
	*dbPassword = envOrDefault(*dbPassword, "", "ISC_DB_PASSWORD")
	*dbSSLMode = envOrDefault(*dbSSLMode, "disable", "ISC_DB_SSLMODE")

	if *dbName == "" {
		log.Fatal("database name is required: set -db-name or ISC_DB_NAME")
	}
	if *dbUser == "" {
		log.Fatal("database user is required: set -db-user or ISC_DB_USER")
	}
	if *dbPassword == "" {
		log.Fatal("database password is required: set -db-password or ISC_DB_PASSWORD")
	}

	dbPortInt, err := strconv.Atoi(*dbPort)
	if err != nil {
		log.Fatalf("invalid db-port: %v", err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := config.Validate(cfg); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	db, err := database.Connect(*dbHost, dbPortInt, *dbName, *dbUser, *dbPassword, *dbSSLMode)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, cfg.Tables); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	fmt.Println("Migrations complete")
}
