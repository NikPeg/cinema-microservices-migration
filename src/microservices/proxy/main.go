package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the proxy configuration
type Config struct {
	Port                   string
	MonolithURL           string
	MoviesServiceURL      string
	EventsServiceURL      string
	GradualMigration      bool
	MoviesMigrationPercent int
}

// ProxyServer represents the proxy server
type ProxyServer struct {
	config           *Config
	monolithProxy    *httputil.ReverseProxy
	moviesProxy      *httputil.ReverseProxy
	eventsProxy      *httputil.ReverseProxy
	rand             *rand.Rand
}

// NewProxyServer creates a new proxy server instance
func NewProxyServer(config *Config) (*ProxyServer, error) {
	monolithURL, err := url.Parse(config.MonolithURL)
	if err != nil {
		return nil, fmt.Errorf("invalid monolith URL: %w", err)
	}

	moviesURL, err := url.Parse(config.MoviesServiceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid movies service URL: %w", err)
	}

	eventsURL, err := url.Parse(config.EventsServiceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid events service URL: %w", err)
	}

	return &ProxyServer{
		config:        config,
		monolithProxy: createReverseProxy(monolithURL),
		moviesProxy:   createReverseProxy(moviesURL),
		eventsProxy:   createReverseProxy(eventsURL),
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

// createReverseProxy creates a reverse proxy with custom director
func createReverseProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Custom director to preserve the original path
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.URL.Host = target.Host
		req.URL.Scheme = target.Scheme
	}

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Service temporarily unavailable",
		})
	}

	return proxy
}

// ServeHTTP handles incoming HTTP requests
func (ps *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	log.Printf("Incoming request: %s %s", r.Method, path)

	// Handle health check
	if path == "/health" {
		ps.handleHealth(w, r)
		return
	}

	// Route events service requests
	if strings.HasPrefix(path, "/api/events") {
		log.Printf("Routing to events service: %s", path)
		ps.eventsProxy.ServeHTTP(w, r)
		return
	}

	// Route movies requests with gradual migration
	if strings.HasPrefix(path, "/api/movies") {
		if ps.shouldRouteToNewService() {
			log.Printf("Routing to movies microservice: %s (migration: %d%%)",
				path, ps.config.MoviesMigrationPercent)
			ps.moviesProxy.ServeHTTP(w, r)
		} else {
			log.Printf("Routing to monolith: %s (migration: %d%%)",
				path, ps.config.MoviesMigrationPercent)
			ps.monolithProxy.ServeHTTP(w, r)
		}
		return
	}

	// Route all other requests to monolith
	log.Printf("Routing to monolith (default): %s", path)
	ps.monolithProxy.ServeHTTP(w, r)
}

// shouldRouteToNewService determines if request should go to new service based on migration percentage
func (ps *ProxyServer) shouldRouteToNewService() bool {
	if !ps.config.GradualMigration {
		// If gradual migration is disabled, always route to new service
		return true
	}

	// Use random number to determine routing based on percentage
	randomValue := ps.rand.Intn(100)
	return randomValue < ps.config.MoviesMigrationPercent
}

// handleHealth handles health check endpoint
func (ps *ProxyServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Strangler Fig Proxy is healthy"))
}

// loadConfig loads configuration from environment variables
func loadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	monolithURL := os.Getenv("MONOLITH_URL")
	if monolithURL == "" {
		monolithURL = "http://localhost:8080"
	}

	moviesServiceURL := os.Getenv("MOVIES_SERVICE_URL")
	if moviesServiceURL == "" {
		moviesServiceURL = "http://localhost:8081"
	}

	eventsServiceURL := os.Getenv("EVENTS_SERVICE_URL")
	if eventsServiceURL == "" {
		eventsServiceURL = "http://localhost:8082"
	}

	gradualMigration := os.Getenv("GRADUAL_MIGRATION") == "true"

	migrationPercent := 50
	if percentStr := os.Getenv("MOVIES_MIGRATION_PERCENT"); percentStr != "" {
		if percent, err := strconv.Atoi(percentStr); err == nil {
			migrationPercent = percent
		}
	}

	return &Config{
		Port:                   port,
		MonolithURL:           monolithURL,
		MoviesServiceURL:      moviesServiceURL,
		EventsServiceURL:      eventsServiceURL,
		GradualMigration:      gradualMigration,
		MoviesMigrationPercent: migrationPercent,
	}
}

// LoggingMiddleware logs all incoming requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture status code
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)
		log.Printf("[%s] %s %s - Status: %d - Duration: %v",
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
			lrw.statusCode,
			duration,
		)
	})
}

// loggingResponseWriter wraps http.ResponseWriter to capture status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Load configuration
	config := loadConfig()

	// Log configuration
	log.Printf("Starting Strangler Fig Proxy Server")
	log.Printf("Port: %s", config.Port)
	log.Printf("Monolith URL: %s", config.MonolithURL)
	log.Printf("Movies Service URL: %s", config.MoviesServiceURL)
	log.Printf("Events Service URL: %s", config.EventsServiceURL)
	log.Printf("Gradual Migration: %v", config.GradualMigration)
	log.Printf("Movies Migration Percentage: %d%%", config.MoviesMigrationPercent)

	// Create proxy server
	proxyServer, err := NewProxyServer(config)
	if err != nil {
		log.Fatalf("Failed to create proxy server: %v", err)
	}

	// Setup middleware chain
	handler := LoggingMiddleware(CORSMiddleware(proxyServer))

	// Start HTTP server
	addr := ":" + config.Port
	log.Printf("Proxy server listening on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
