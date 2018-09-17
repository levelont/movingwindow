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

//of course that currentRequestCount has an additional field that could be used for the global accumulatedRequestCount
// however, that would be mixing up different concerns: the persistence.RequestCount.accumulatedRequestCount field is meant for
//internal use of the doublylinkedlist only, so that it can calculate the current accumulated values for the relevant time frame
// and therefore not exported.
//thus, a specific counter is added to the request cache.
//TODO names should be used for this differentiation.
type requestCount struct {
	persistence.RequestCount
	AccumulatedRequestCount int
}

//TODO name
//this is the information that the persistence processor requires: a requestCount and a reference timestamp
type persistenceData struct {
	requestCount persistence.RequestCount
	reference    persistence.RequestCount
}

//TODO variable names - the type name is also silly
type communication struct {
	cache                requestCount
	persistedObjects     persistence.RequestCountDoublyLinkedList
	exchangeTimestamp    chan time.Time
	exchangeRequestCount chan requestCount
	exchangePersistence  chan persistenceData
	exchangeAccumulated  chan int
}

func NewCommunication() communication {
	return communication{
		exchangeTimestamp:    make(chan time.Time),
		exchangeRequestCount: make(chan requestCount),
		exchangePersistence:  make(chan persistenceData),
		exchangeAccumulated:  make(chan int),
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
		//TODO close all channels
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

func (s *server) startCommunicationProcessor() {
	s.logger.Print("Starting communication processor...")

	s.logger.Print("Starting Persistence-Accumulated exchanger...")
	go func(com communication) {
		for {
			persistenceData, ok := <-com.exchangePersistence
			if ok {
				com.persistedObjects = com.persistedObjects.AppendToTail(persistenceData.requestCount)
				com.persistedObjects = com.persistedObjects.UpdateTotals(persistenceData.reference)
				com.exchangeAccumulated <- com.persistedObjects.TotalAccumulatedRequestCount()
			} else {
				break
			}
		}
	}(s.communication)

	s.logger.Print("Starting Timestamp-RequestCount exchanger...")
	go func(com communication) {
		for {
			requestTimestamp, ok := <-com.exchangeTimestamp
			if ok {
				s.logger.Printf("COM: received new requestTimestamp: '%v'\n", requestTimestamp.Format(time.RFC3339))

				//TODO lazy initialization with Sync.once
				if com.cache.Empty() {
					com.cache.Timestamp = requestTimestamp
					com.cache.AccumulatedRequestCount = 0
					s.logger.Print("COM: Initialized cache")
				}

				if com.cache.CompareTimestampWithPrecision(requestTimestamp, time.Second) {
					com.cache.RequestsCount = com.cache.RequestsCount + 1
					com.cache.AccumulatedRequestCount = com.cache.RequestsCount
					s.logger.Printf("COM: Incremented cached requestTimestamp to '%v'\n", com.cache.RequestsCount)
				} else {
					//timestamps are different.
					//send current cache to the persistence goroutine
					//it gives back the total for last 60s
					//TODO

					persistenceUpdate := persistenceData{
						requestCount: com.cache.RequestCount,
						reference: persistence.RequestCount{
							Timestamp: requestTimestamp,
						},
					}

					s.communication.exchangePersistence <- persistenceUpdate
					totalAccumulated := <-s.communication.exchangeAccumulated

					com.cache = requestCount{
						RequestCount: persistence.RequestCount{
							Timestamp:     requestTimestamp,
							RequestsCount: 1,
						},
						AccumulatedRequestCount: totalAccumulated,
					}
				}

				//there is only one value that needs be returned: the total count.
				//requestCache is used. Because only its accumulate field is exported,
				//only it will be marshalled.
				com.exchangeRequestCount <- com.cache
			} else {
				break
			}
		}
	}(s.communication)
	s.logger.Print("Communication processor up and running")
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
