package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"steam-profit-calculator/internal/assistant"
	"steam-profit-calculator/internal/lookup"
	"steam-profit-calculator/internal/storage"
	"steam-profit-calculator/internal/webapi"
)

const (
	defaultAddress      = "127.0.0.1:8080"
	defaultDatabaseFile = "steam-profit.db"
	defaultRecordFile   = "steam-profit-records.csv"
)

func main() {
	address := flag.String("addr", environmentOrDefault("STEAM_PROFIT_SERVER_ADDR", defaultAddress), "HTTP listen address")
	storageMode := flag.String("storage", environmentOrDefault("STEAM_PROFIT_STORAGE", "sqlite"), "record storage: sqlite or csv")
	databaseFile := flag.String("db", environmentOrDefault("STEAM_PROFIT_DATABASE_FILE", defaultDatabaseFile), "SQLite database path")
	recordFile := flag.String("file", environmentOrDefault("STEAM_PROFIT_RECORD_FILE", defaultRecordFile), "legacy CSV record path used by CSV storage")
	importCSV := flag.String("import-csv", environmentOrDefault("STEAM_PROFIT_IMPORT_CSV", ""), "compatible CSV file to import into SQLite before startup")
	flag.Parse()

	var recordStore storage.RecordRepository
	switch strings.ToLower(strings.TrimSpace(*storageMode)) {
	case "sqlite":
		repository, err := storage.OpenSQLiteRepository(*databaseFile)
		if err != nil {
			log.Fatalf("open SQLite record repository: %v", err)
		}
		defer func() {
			if err := repository.Close(); err != nil {
				log.Printf("close SQLite record repository: %v", err)
			}
		}()
		if path := strings.TrimSpace(*importCSV); path != "" {
			result, err := repository.ImportCSV(context.Background(), path)
			if err != nil {
				log.Fatalf("import CSV records: %v", err)
			}
			log.Printf("CSV import complete: %d inserted, %d skipped", result.Inserted, result.Skipped)
		}
		recordStore = repository
		log.Printf("record storage: SQLite (%s)", *databaseFile)
	case "csv":
		if strings.TrimSpace(*importCSV) != "" {
			log.Fatal("--import-csv requires --storage=sqlite")
		}
		recordStore = storage.NewCSVRepository(*recordFile)
		log.Printf("record storage: legacy CSV (%s)", *recordFile)
	default:
		log.Fatalf("invalid record storage %q: expected sqlite or csv", *storageMode)
	}

	lookupClient, err := lookup.NewClientFromEnvironment()
	if err != nil {
		log.Fatalf("load lookup configuration: %v", err)
	}
	model := assistant.NewModelFromEnvironment()
	toolRunner, err := assistant.NewToolRunner(lookupClient)
	if err != nil {
		log.Fatalf("create assistant tools: %v", err)
	}
	assistantService, err := assistant.NewService(model, toolRunner)
	if err != nil {
		log.Fatalf("create assistant: %v", err)
	}
	handler, err := webapi.NewHandlerWithAssistant(lookupClient, recordStore, assistantService)
	if err != nil {
		log.Fatalf("create web application: %v", err)
	}

	httpServer := &http.Server{
		Addr:              *address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      70 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-shutdownContext.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("Steam Profit web application available at http://%s", displayAddress(*address))
	if status := assistantService.Status(); status.Available {
		log.Printf("AI strategy assistant enabled with %s/%s (%s)", status.Provider, status.Model, status.PromptVersion)
	} else {
		log.Printf("AI strategy assistant unavailable: %s", status.Reason)
	}
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve web application: %v", err)
	}
}

func environmentOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func displayAddress(address string) string {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}
