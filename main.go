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

//TODO variable names - the type name is also silly
type communication struct {
	state                persistence.State
	exchangeTimestamp    chan time.Time
	exchangeRequestCount chan persistence.Cache
	exchangePersistence  chan persistence.PersistenceData
	exchangeAccumulated  chan int
}

//TODO document purpose of each communication channel
func NewCommunication() communication {
	return communication{
		exchangeTimestamp:    make(chan time.Time),
		exchangeRequestCount: make(chan persistence.Cache),
		exchangePersistence:  make(chan persistence.PersistenceData),
		exchangeAccumulated:  make(chan int),
	}
}

type server struct {
	router               *http.ServeMux
	logger               *log.Logger
	communication        communication
	persistenceTimeFrame time.Duration
	persistenceFile      string
	http.Server
}

type environment struct {
	listenAddress              string
	persistenceFile            string
	persistenceTimeFrame       string
	parsedPersistenceTimeFrame time.Duration
}

func parseEnvironment() environment {
	var env environment
	flag.StringVar(&env.listenAddress, "listen-address", ":5000", "Server listen address")
	flag.StringVar(&env.persistenceTimeFrame, "persistence-time-interval", "60s", "Time frame for which request counts will be calculated")
	flag.StringVar(&env.persistenceFile, "persistence-file", "persistence.bin", "File to which state will be persisted upon server termination")
	flag.Parse()

	var err error
	env.parsedPersistenceTimeFrame, err = time.ParseDuration(env.persistenceTimeFrame)
	if err != nil {
		panic(err) //OK: need env variable to be parsable.
	}

	return env
}

func NewServer(env environment) *server {
	router := http.NewServeMux()
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	errorLogger := log.New(os.Stderr, "http: ", log.LstdFlags)
	communication := NewCommunication()
	server := &server{
		router:               router,
		logger:               logger,
		communication:        communication,
		persistenceTimeFrame: env.parsedPersistenceTimeFrame,
		persistenceFile:      env.persistenceFile,
		Server: http.Server{
			Addr:         env.listenAddress,
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
	server := NewServer(parseEnvironment())
	server.logger.Println("Server is starting...")
	server.routes()

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		signal := <-quit
		server.logger.Printf("Server received signal '%v'. Saving state to file '%v'\n", signal, server.persistenceFile)

		if err := server.communication.state.WriteToFile(server.persistenceFile); err != nil {
			server.logger.Fatalf("Could not save state to disk: %v\n", err)
		}

		server.logger.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			server.logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		//TODO close all channels
		close(done)
	}()

	server.logger.Println("Server is ready to handle requests at", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		server.logger.Fatalf("Could not listen on %s: %v\n", server.Addr, err)
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
				//TODO workflow wrapper?
				com.state.Past = com.state.Past.AppendToTail(persistenceData.RequestCount)
				com.state.Past = com.state.Past.UpdateTotals(persistenceData.Reference, s.persistenceTimeFrame)
				com.exchangeAccumulated <- com.state.Past.TotalAccumulatedRequestCount()
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

				if com.state.Present.Empty() {
					com.state.Present.Timestamp = requestTimestamp
					s.logger.Print("COM: Initialized cache")
				}

				if com.state.Present.CompareTimestampWithPrecision(requestTimestamp, time.Second) {
					com.state.Present.Increment()
					s.logger.Printf("COM: Incremented cached requestTimestamp to '%v'\n", com.state.Present.RequestsCount)
				} else {
					//timestamps are different.
					//send current cache to the persistence goroutine
					//it gives back the total for last 60s
					//TODO

					persistenceUpdate := persistence.NewPersistenceData(com.state.Present, requestTimestamp)
					s.logger.Printf("COM: Sending persistence Update :'%v'\n", persistenceUpdate)

					s.communication.exchangePersistence <- persistenceUpdate
					totalAccumulated := <-s.communication.exchangeAccumulated

					s.logger.Printf("COM: Received new total accumulate of '%v'\n", totalAccumulated)

					com.state.Present = persistence.NewCache(requestTimestamp, totalAccumulated)
					s.logger.Printf("COM: Updated cache to '%v'\n", com.state.Present)
				}

				//there is only one value that needs be returned: the total count.
				//requestCache is used. Because only its accumulate field is exported,
				//only it will be marshalled.
				com.exchangeRequestCount <- com.state.Present
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
