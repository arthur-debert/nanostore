package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	// Global loggers
	mainLogger    *slog.Logger
	queriesLogger *slog.Logger
	resultsLogger *slog.Logger

	// Log level mapping
	logLevelMap = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
)

// initLogging initializes the logging system with file and optional stdout handlers
func initLogging(logLevel string, logQueries bool, logResults bool) error {
	// Parse log level
	level, ok := logLevelMap[strings.ToLower(logLevel)]
	if !ok {
		level = slog.LevelWarn // Default to WARN
	}

	// Get XDG cache directory
	logDir := getXDGCacheDir()
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file
	logPath := filepath.Join(logDir, "nano-db.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create file handler with JSON format for structured logging
	fileHandler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})

	// Create main logger
	mainLogger = slog.New(fileHandler)
	slog.SetDefault(mainLogger)

	// Create queries logger
	queriesLogPath := filepath.Join(logDir, "nano-db-queries.log")
	queriesLogFile, err := os.OpenFile(queriesLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open queries log file: %w", err)
	}

	// Base handler for queries is always the file
	var queriesHandler slog.Handler = slog.NewJSONHandler(queriesLogFile, &slog.HandlerOptions{
		Level: slog.LevelInfo, // Always log queries at INFO level
	})

	// If logQueries is true, also log to stdout
	if logQueries {
		// Create a multi-handler that logs to both file and stdout
		stdoutHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		queriesHandler = &multiHandler{
			handlers: []slog.Handler{queriesHandler, stdoutHandler},
		}
	}

	queriesLogger = slog.New(queriesHandler).With("logger", "queries")

	// Create results logger (for logging operation results)
	resultsLogPath := filepath.Join(logDir, "nano-db-results.log")
	resultsLogFile, err := os.OpenFile(resultsLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open results log file: %w", err)
	}

	// Base handler for results is always the file
	var resultsHandler slog.Handler = slog.NewJSONHandler(resultsLogFile, &slog.HandlerOptions{
		Level: slog.LevelInfo, // Always log results at INFO level
	})

	// If logResults is true, also log to stdout
	if logResults {
		// Create a multi-handler that logs to both file and stdout
		stdoutHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		resultsHandler = &multiHandler{
			handlers: []slog.Handler{resultsHandler, stdoutHandler},
		}
	}

	resultsLogger = slog.New(resultsHandler).With("logger", "results")

	mainLogger.Debug("logging initialized",
		"level", level.String(),
		"log_file", logPath,
		"queries_file", queriesLogPath,
		"results_file", resultsLogPath,
		"log_queries_stdout", logQueries,
		"log_results_stdout", logResults)

	return nil
}

// getXDGCacheDir returns the XDG cache directory for nano-db
func getXDGCacheDir() string {
	// First check XDG_CACHE_HOME
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		return filepath.Join(xdgCache, "nano-db")
	}

	// Fall back to default based on OS
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Last resort - use temp directory
		return filepath.Join(os.TempDir(), "nano-db")
	}

	if runtime.GOOS == "darwin" {
		// macOS uses ~/Library/Caches
		return filepath.Join(homeDir, "Library", "Caches", "nano-db")
	}

	// Linux and others use ~/.cache
	return filepath.Join(homeDir, ".cache", "nano-db")
}

// multiHandler implements slog.Handler to write to multiple handlers
type multiHandler struct {
	handlers []slog.Handler
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: newHandlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: newHandlers}
}

// logSQLQuery logs an actual SQL query sent to the database
func logSQLQuery(operation string, sql string, args []interface{}) {
	if queriesLogger != nil {
		queriesLogger.Info("sql_query",
			"operation", operation,
			"sql", sql,
			"args", args,
		)
	}
}

// logOperation logs an operation result (what was previously logged by logQuery)
func logOperation(operation string, description string, args []interface{}) {
	if resultsLogger != nil {
		resultsLogger.Info("operation",
			"operation", operation,
			"description", description,
			"args", args,
		)
	}
}
