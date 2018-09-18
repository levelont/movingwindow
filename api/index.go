package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

/*
Received value from the exchangeRequestCount will be wrapped in a 'response' type to hide information that should
not make it to the client. Information hiding is the main purpose of this struct.
Exported for tests to consume
*/
type Response struct {
	timestamp    time.Time
	RequestCount int `json:"requestCount"`
}

/* Allows for construction with specification of the timestamp member while avoid exporting of the timestamp
field, which needs to be hidden from the client.
*/
func NewResponse(timestamp time.Time, requestCount int) Response {
	return Response{timestamp: timestamp, RequestCount: requestCount}
}

/* Provide access to the timestamp field while avoiding to export it.
 */
func (r Response) Timestamp() time.Time {
	return r.timestamp
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

/* Handler hangs on the server so that it can access to all communication and persistence variables.
Variables are passed to the handler function in a closure fashion. Updating the communication values
on the server will therefore have no effect in it's functionality.
See communication::StartCommunicationProcessor for documentation on the workflow.
Delaying init until the handler is actually called via sync.Once saves on http server boot up time.
*/
func (s *server) Index(com communication) http.HandlerFunc {
	var init sync.Once
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		init.Do(s.initialize)

		if r.URL.Path != "/" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		requestTimestamp := time.Now().Truncate(time.Second)
		s.Logger.Printf("RequestTimestamp: '%v'\n", requestTimestamp.Format(time.RFC3339))

		com.exchangeTimestamp <- requestTimestamp
		totalRequestsSoFar := <-com.exchangeRequestCount
		s.Logger.Printf("Received cache from communication processor: '%v'\n", totalRequestsSoFar)

		response := Response{
			timestamp:    totalRequestsSoFar.Timestamp,
			RequestCount: totalRequestsSoFar.TotalRequestsWithinTimeframe,
		}
		encodedCache, err := json.Marshal(response)
		if err != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, ResponseError{errorMsg: err.Error()}.ToJSON())
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(encodedCache))
		s.Logger.Printf("Done")
	})
}
