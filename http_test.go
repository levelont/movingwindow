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
type requestCountList []cache

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
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
		},
		sorted: requestCountList{
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
		},
	},

	{ // Timestamp collisions
		unsorted: requestCountList{
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 3}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 1}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
		},
		sorted: requestCountList{
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 1}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 3}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
		},
	},
	{ // Timestamp and request count collisions
		unsorted: requestCountList{
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 3}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 1}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
		},
		sorted: requestCountList{
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 1}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 3}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
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

//parms for index testing
//'intervals' for requests and ammount of requests to be sent for each of em
//total requests at the end

type requestsOrder struct {
	numRequests int
	delay       int
	unit        time.Duration
}

type indexHandleTest struct {
	requestList []requestsOrder
	total       int
}

var indexHandleTestList = []indexHandleTest{
	{ //single instant, 1 requests
		requestList: []requestsOrder{
			{numRequests: 1, delay: 0, unit: time.Second},
		},
		total: 1,
	},
	{ //two instants, 1s=2 2s=2
		requestList: []requestsOrder{
			{numRequests: 2, delay: 0, unit: time.Second},
			{numRequests: 2, delay: 1, unit: time.Second},
		},
		total: 4,
	},
	{ //single instant, 1000 requests
		requestList: []requestsOrder{
			{numRequests: 1000, delay: 0, unit: time.Second},
		},
		total: 1000,
	},
	{ // three consecutive instants, 1s=1000, 2s=1000, 3s=1000
		requestList: []requestsOrder{
			{numRequests: 1000, delay: 0, unit: time.Second},
			{numRequests: 1000, delay: 1, unit: time.Second},
			{numRequests: 1000, delay: 1, unit: time.Second},
		},
		total: 3000,
	},
	{ // five instants with long delays in between, 1s=1000, 2s=1000, 3s=1000
		requestList: []requestsOrder{
			{numRequests: 1000, delay: 0, unit: time.Second},
			{numRequests: 1000, delay: 1, unit: time.Second},
			{numRequests: 1000, delay: 1, unit: time.Second},
		},
		total: 3000,
	},
}

func TestHandleIndex(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal("Wicked error while creating a new request")
	}

	dispatcherToAppender := make(chan cache)
	appenderToDispatcher := make(chan bool)
	done := make(chan bool)
	//only need one processor. These can be started before

	var responses requestCountList
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

	//TODO names
	//TODO hideous indentation
	var w *httptest.ResponseRecorder
	for i, test := range indexHandleTestList {
		srv := NewServer(":5000", time.Duration(60)*time.Second)
		//srv.logger.SetOutput(ioutil.Discard)
		srv.routes()
		handler := srv.index(srv.communication)
		responses = make(requestCountList, 0, test.total)

		for _, order := range test.requestList {
			time.Sleep(time.Duration(order.delay) * order.unit)

			for i := 0; i < order.numRequests; i++ { // individual num requests of each test's request
				w = httptest.NewRecorder()

				go func(w *httptest.ResponseRecorder) {
					handler(w, req)

					if w.Result().StatusCode != http.StatusOK {
						t.Fatalf("Something went wrong with request '%v'\n", req)
					}

					buf := new(bytes.Buffer)
					buf.ReadFrom(w.Result().Body)

					var receivedResponse cache
					json.Unmarshal(buf.Bytes(), &receivedResponse)
					log.Printf("DISPATCHER: Received response: '%v'\n", receivedResponse)
					dispatcherToAppender <- receivedResponse
					<-appenderToDispatcher
					log.Print("DISPATCHER: Finished processing.")
					done <- true
				}(w)
			}
		}

		for i := 0; i < test.total; i++ {
			<-done
		}

		if len(responses) != test.total {
			t.Errorf("Expected to receive '%v' response for test '%v', but got '%v' instead.\n", test.total, i, len(responses))
		}

		sort.Sort(responses)
		for index, response := range responses {
			if response.AccumulatedRequestCount != index+1 {
				t.Errorf("Expected received response '%v' to have a count of '%v' but didn't. Test; '%v', Values: '%v'\n", response, index+1, i, test)
			}
		}
	}

	//current approach is that every response will be an item in the 'responses' list
	// and that the number of requests will match the index
	//what this means is that each and every request will get the requestcount as response in the order in which they arrive,
	//and that no requests are lost. nifty!

	close(done)
	close(dispatcherToAppender)
	close(appenderToDispatcher)
}
