package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"simplesurance/persistence"
	"sync"
	"time"
)

type key int

const (
	requestIDKey key = 0
)

type communication struct {
	cache                persistence.RequestCount
	exchangeTimestamp    chan time.Time
	exchangeRequestCount chan persistence.RequestCount
}

func NewCommunication() communication {
	return communication{
		exchangeTimestamp:    make(chan time.Time),
		exchangeRequestCount: make(chan persistence.RequestCount),
	}
}

var (
	listenAddress string
)

type server struct {
	router        *http.ServeMux
	logger        *log.Logger
	communication communication
	http.Server
}

func NewServer(listenAddress string) *server {
	router := http.NewServeMux()
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	errorLogger := log.New(os.Stderr, "http: ", log.LstdFlags)
	communication := NewCommunication()
	server := &server{
		router:        router,
		logger:        logger,
		communication: communication,
		Server: http.Server{
			Addr:         listenAddress,
			Handler:      tracing(nextRequestID)(logging(logger)(router)),
			ErrorLog:     errorLogger,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
	}

	return server
}

func (s *server) routes() {
	s.router.HandleFunc("/", s.index(s.communication))
}

func nextRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func (s *server) startCommunicationProcessor() {
	s.logger.Print("Starting communication processor...")
	go func(com communication) {
		for {
			requestTimestamp := <-com.exchangeTimestamp
			s.logger.Printf("COM: received new requestTimestamp: '%v'\n", requestTimestamp.Format(time.RFC3339))

			//TODO lazy initialization with Sync.once
			if com.cache.Empty() {
				com.cache.Timestamp = requestTimestamp
				s.logger.Print("COM: Initialized cache")
			}

			if com.cache.CompareTimestampWithPrecision(requestTimestamp, time.Second) {
				com.cache.RequestsCount = com.cache.RequestsCount + 1
				s.logger.Printf("COM: Incremented cached requestTimestamp to '%v'\n", com.cache.RequestsCount)
			}

			com.exchangeRequestCount <- com.cache
		}
	}(s.communication)
	s.logger.Print("Communication processor up and running")
}

func main() {
	flag.StringVar(&listenAddress, "listen-addr", ":5000", "server listen address")
	flag.Parse()

	server := NewServer(listenAddress)
	server.logger.Println("Server is starting...")
	server.routes()

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		server.logger.Println("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			server.logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	server.logger.Println("Server is ready to handle requests at", listenAddress)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		server.logger.Fatalf("Could not listen on %s: %v\n", listenAddress, err)
	}

	<-done
	server.logger.Println("Server stopped")
}

type ResponseError struct {
	errorMsg string
}

func (r ResponseError) ToJSON() string {
	encodedError, err := json.Marshal(r)
	if err != nil {
		log.Fatal(err)
	}
	return string(encodedError)
}

func (s *server) index(com communication) http.HandlerFunc {
	var (
		init sync.Once
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		init.Do(s.startCommunicationProcessor)

		if r.URL.Path != "/" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		//get timestamp, truncate to seconds
		requestTimestamp := time.Now().Truncate(time.Second)
		s.logger.Printf("RequestTimestamp: '%v'\n", requestTimestamp.Format(time.RFC3339))

		com.exchangeTimestamp <- requestTimestamp
		totalRequestsSoFar := <-com.exchangeRequestCount
		s.logger.Printf("Received cache from communication processor: '%v'\n", totalRequestsSoFar)

		encodedCache, err := json.Marshal(totalRequestsSoFar)
		if err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, ResponseError{errorMsg: err.Error()}.ToJSON())
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(encodedCache))
		s.logger.Printf("Done")
	})
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
