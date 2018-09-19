package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"simplesurance/persistence"
	"time"
)

type key int

const (
	requestIDKey key = 0
)

func nextRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

/* Wrapper for all information required in the handler.
 */
type server struct {
	router               *http.ServeMux
	Logger               *log.Logger
	Communication        communication
	persistenceTimeFrame time.Duration
	precision            time.Duration
	persistenceFile      string
	http.Server
}

func NewServer(env Environment) *server {
	router := http.NewServeMux()
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	errorLogger := log.New(os.Stderr, "http: ", log.LstdFlags)
	communication := NewCommunication()
	server := &server{
		router:               router,
		Logger:               logger,
		Communication:        communication,
		persistenceTimeFrame: env.PersistenceTimeFrame,
		precision:            env.Precision,
		persistenceFile:      env.PersistenceFile,
		Server: http.Server{
			Addr:         env.ListenAddress,
			Handler:      tracing(nextRequestID)(logging(logger)(router)),
			ErrorLog:     errorLogger,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
	}

	return server
}

/*Read persisted state from disk during startup. Upon successful read, the file will be removed.
Keep in mind: by the time the server has been restarted, the persisted values read and a new request is to be handled, the
persisted request counts might no longer be in the persistence time frame. In that case, they will be discarded for the
next request count computation.
*/
func (s *server) readStateFromDisk() {
	if _, err := os.Open(s.persistenceFile); err != nil {
		s.Logger.Printf("No state file could be found under '%v': %v. Will work on a clean slate.\n", s.persistenceFile, err)
	} else {
		s.Logger.Printf("Reading last state from file '%v'...\n", s.persistenceFile)
		s.Communication.state, err = persistence.ReadFromFile(s.persistenceFile)
		if err != nil {
			s.Logger.Printf("Could not read state from file '%v': %v\n", s.persistenceFile, err)
			return
		}
		s.Logger.Printf("State restored. Current request count: '%v'\n", s.Communication.state.Present.TotalRequestsWithinTimeframe)

		s.Logger.Println("Removing file...")
		if err := os.Remove(s.persistenceFile); err != nil {
			s.Logger.Printf("Could not remove state file '%v': %v\n", s.persistenceFile, err)
		}
	}
}

func (s *server) PersistState() error {
	s.Logger.Printf("Persisting state '%+v' to file '%v'.", s.Communication.state, s.persistenceFile)
	if err := s.Communication.state.WriteToFile(s.persistenceFile); err != nil {
		return err
	}
	return nil
}

func (s *server) initialize() {
	s.Logger.Print("Initialising server with following parameters:")
	s.Logger.Printf("Persistence Timeframe: '%v'\n", s.persistenceTimeFrame)
	s.Logger.Printf("Persistence File: '%v'\n", s.persistenceFile)
	s.readStateFromDisk()
	s.startCommunicationProcessor()
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
