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
	"testing"
	"time"
)

//use an array to store requestCounts
//the result of the request is a marshalled JSON that we can unmarshall into a requestCount object
//we just need the structure to implement sort interface with
//TODO move these variables and the test somewhere else? or just document properly.
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
	srv := NewServer(":5000")
	//srv.logger.SetOutput(ioutil.Discard)
	srv.routes()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal("Wicked error while creating a new request")
	}

	handler := srv.index(srv.communication)

	numRequests := 1000
	responses := make(requestCountList, 0, numRequests)
	//TODO names
	dispatcherToAppender := make(chan persistence.RequestCount)
	appenderToDispatcher := make(chan bool)
	done := make(chan bool)

	go func() {
		for {
			log.Println("APPENDER: Waiting for response...")
			receivedResponse, ok := <-dispatcherToAppender
			if ok {
				responses = append(responses, receivedResponse)
				log.Printf("APPENDER: Successfully added response '%v'\n", receivedResponse)
				appenderToDispatcher <- true
			} else {
				break
			}
		}
	}()

	var w *httptest.ResponseRecorder
	for i := 0; i < numRequests; i++ {
		w = httptest.NewRecorder()
		go func(w *httptest.ResponseRecorder) {
			handler(w, req)

			buf := new(bytes.Buffer)
			buf.ReadFrom(w.Result().Body)

			var receivedResponse persistence.RequestCount
			json.Unmarshal(buf.Bytes(), &receivedResponse)
			log.Printf("DISPATCHER: Received response: '%v'\n", receivedResponse)
			dispatcherToAppender <- receivedResponse
			<-appenderToDispatcher
			log.Print("DISPATCHER: Finished processing.")
			done <- true
		}(w)
	}

	for i := 0; i < numRequests; i++ {
		<-done
	}
	close(done)
	close(dispatcherToAppender)
	close(appenderToDispatcher)

	//TODO check status of all responses, not just the last one.
	if w.Result().StatusCode != http.StatusOK {
		t.Error("Something went wrong")
	}

	if len(responses) != numRequests {
		t.Errorf("Expected to receive '%v' responses, but got '%v' instead.\n", len(responses), numRequests)
	}

	sort.Sort(responses)
	for index, response := range responses {
		if response.RequestsCount != index+1 {
			t.Errorf("Expected received response '%v' to have a count of '%v' but didn't.\n", response, index+1)
		}
	}
}
