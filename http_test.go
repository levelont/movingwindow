package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"simplesurance/persistence"
	"sort"
	"sync"
	"testing"
	"time"
)

//use an array to store requestCounts
//the result of the request is a marshalled JSON that we can unmarshall into a requestCount object
//we just need the structure to implement sort interface with

type requestCountList []persistence.RequestCount

func (s requestCountList) Len() int {
	return len(s)
}
func (s requestCountList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s requestCountList) Less(i, j int) bool {
	if s[i].CompareTimestampWithPrecision(s[j].Timestamp, time.Second) {
		return s[i].RequestsCount < s[j].RequestsCount
	} else {
		return s[i].Timestamp.Before(s[j].Timestamp)
	}
}

type requestCountListSortingTest struct {
	unsorted requestCountList
	sorted   requestCountList
}

var requestCountListSortingTestList = []requestCountListSortingTest{
	{ // Basic test
		unsorted: requestCountList{
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)},
		},
		sorted: requestCountList{
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)},
		},
	},

	{ // Timestamp collisions
		unsorted: requestCountList{
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 3},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 1},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)},
		},
		sorted: requestCountList{
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 1},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 3},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)},
		},
	},
	{ // Timestamp and request count collisions
		unsorted: requestCountList{
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 3},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 1},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)},
		},
		sorted: requestCountList{
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 1},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 3},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)},
			persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)},
		},
	},
}

func TestRequestCountListSorting(t *testing.T) {
	for i, test := range requestCountListSortingTestList {
		sort.Sort(test.unsorted)
		if !reflect.DeepEqual(test.unsorted, test.sorted) {
			t.Fatalf("Expected list to be sorted after calling sort on it, but it didn't match the expected values. Test: '%v', Data: '%v'\n", i, test)
		}
	}
}

func TestHandleAbout(t *testing.T) {
	log.Println("Is this turned on?")
	srv := NewServer(":5000")
	//srv.logger.SetOutput(ioutil.Discard)
	srv.routes()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal("Wicked error while creating a new request")
	}

	handler := srv.index(srv.communication)

	var wg sync.WaitGroup

	numRequests := 3
	responses := make(requestCountList, 0, numRequests)
	responseReady := make(chan persistence.RequestCount)

	go func(r requestCountList, rc chan persistence.RequestCount) {
		for {
			log.Println("APPENDER: Waiting for response...")
			receivedResponse := <-rc
			r = append(r, receivedResponse)
			log.Printf("APPENDER: Successfully added response '%v'\n", receivedResponse)
			rc <- receivedResponse
		}
	}(responses, responseReady)

	var w *httptest.ResponseRecorder
	for i := 0; i < numRequests; i++ {
		w = httptest.NewRecorder()
		wg.Add(1)
		go func(w *httptest.ResponseRecorder, rc chan persistence.RequestCount) {
			handler(w, req)

			buf := new(bytes.Buffer)
			buf.ReadFrom(w.Result().Body)

			var receivedResponse persistence.RequestCount
			json.Unmarshal(buf.Bytes(), &receivedResponse)
			log.Printf("DISPATCHER: Received response: '%v'\n", receivedResponse)
			rc <- receivedResponse
			<-rc
			log.Print("DISPATCHER: Finished processing.")
			wg.Done()
		}(w, responseReady)
	}

	wg.Wait()
	close(responseReady)

	log.Printf("Got '%v' responses: '%v'\n", len(responses), responses)
	if w.Result().StatusCode != http.StatusOK {
		t.Error("Something went wrong")
	}
}

//TODO signal manager MUST close the communication channels!
