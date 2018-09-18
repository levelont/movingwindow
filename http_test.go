package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"simplesurance/persistence"
	"sort"
	"testing"
	"time"
)

type requestList []persistence.Cache

/*The 'duration' value is used to perform timestamp comparisons.
 */
type indexResponse struct {
	response
	precision time.Duration
}

func (ir indexResponse) CompareTimestampWithPrecision(t time.Time, precision time.Duration) bool {
	return ir.Timestamp.Truncate(precision) == t.Truncate(precision)
}

type indexResponseList []indexResponse

func (s indexResponseList) Len() int {
	return len(s)
}
func (s indexResponseList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s indexResponseList) Less(i, j int) bool {
	if s[i].CompareTimestampWithPrecision(s[j].Timestamp, time.Second) {
		return s[i].RequestCount < s[j].RequestCount
	} else {
		return s[i].Timestamp.Before(s[j].Timestamp)
	}
}

type requestCountListSortingTest struct {
	unsorted indexResponseList
	sorted   indexResponseList
}

var requestCountListSortingTestList = []requestCountListSortingTest{
	{ // Basic test
		unsorted: indexResponseList{
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
		},
		sorted: indexResponseList{
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
		},
	},

	{ // Timestamp collisions
		unsorted: indexResponseList{
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 3}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 2}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 1}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
		},
		sorted: indexResponseList{
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 1}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 2}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 3}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
		},
	},
	{ // Timestamp and request count collisions
		unsorted: indexResponseList{
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 3}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 2}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 2}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 1}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
		},
		sorted: indexResponseList{
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 1}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 2}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 2}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestCount: 3}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			indexResponse{response: response{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
		},
	},
}

func TestResponseListSorting(t *testing.T) {
	for i, test := range requestCountListSortingTestList {
		sort.Sort(test.unsorted)
		if !reflect.DeepEqual(test.unsorted, test.sorted) {
			t.Fatalf("Expected list to be sorted after calling sort on it, but it didn't match the expected values. Test: '%v', Data: '%v'\n", i, test)
		}
	}
}

/* An order specifies a number of requests that should be made to the server in a specific point in time, and indicates
the expected values for the corresponding responses. Successive orders will be 'spaced' by a delay, specified in a time
unit. The generated requests for this specific order are expected to have a response count that spans between
requestCountStart and requestCountEnd.
*/
type order struct {
	numRequests       int
	delay             int
	unit              time.Duration
	requestCountStart int
	requestCountEnd   int
}

type indexHandleTest struct {
	orderList            []order
	persistenceTimeframe time.Duration
}

/* After a run is executed, the first assertion made is that just as many responses as there were requests were dispatched
by the server.
*/
func (i indexHandleTest) getTotalRequests() int {
	totalRequests := 0
	for _, order := range i.orderList {
		totalRequests = totalRequests + order.numRequests
	}
	return totalRequests
}

var indexHandleTestList = []indexHandleTest{
	{ //single instant, 1 request, 'infinite' persistenceTimeframe
		orderList: []order{
			{numRequests: 1, delay: 0, unit: time.Second, requestCountStart: 1, requestCountEnd: 1},
		},
		persistenceTimeframe: time.Duration(60) * time.Second,
	},
	{ //two instants, 1s=2 2s=2, 'infinite' persistenceTimeFrame
		orderList: []order{
			{numRequests: 2, delay: 0, unit: time.Second, requestCountStart: 1, requestCountEnd: 2},
			{numRequests: 2, delay: 1, unit: time.Second, requestCountStart: 3, requestCountEnd: 4},
		},
		persistenceTimeframe: time.Duration(60) * time.Second,
	},
	{ //single instant, 1000 requests, 'infinite' persistenceTimeFrame
		orderList: []order{
			{numRequests: 1000, delay: 0, unit: time.Second, requestCountStart: 1, requestCountEnd: 1000},
		},
		persistenceTimeframe: time.Duration(60) * time.Second,
	},
	{ // three consecutive instants, 1s=1000, 2s=1000, 3s=1000, 'infinite' persistenceTimeFrame
		orderList: []order{
			{numRequests: 1000, delay: 0, unit: time.Second, requestCountStart: 1, requestCountEnd: 1000},
			{numRequests: 1000, delay: 1, unit: time.Second, requestCountStart: 1001, requestCountEnd: 2000},
			{numRequests: 1000, delay: 1, unit: time.Second, requestCountStart: 2001, requestCountEnd: 3000},
		},
		persistenceTimeframe: time.Duration(60) * time.Second,
	},
	{ // five instants with delays in between, timeframe of a single second
		orderList: []order{
			{numRequests: 1, delay: 0, unit: time.Second, requestCountStart: 1, requestCountEnd: 1},
			{numRequests: 1, delay: 1, unit: time.Second, requestCountStart: 2, requestCountEnd: 2},
			{numRequests: 2, delay: 1, unit: time.Second, requestCountStart: 2, requestCountEnd: 3},
			{numRequests: 1, delay: 1, unit: time.Second, requestCountStart: 3, requestCountEnd: 3},
			{numRequests: 1, delay: 1, unit: time.Second, requestCountStart: 2, requestCountEnd: 2},
			{numRequests: 1, delay: 2, unit: time.Second, requestCountStart: 1, requestCountEnd: 1},
		},
		persistenceTimeframe: time.Duration(1) * time.Second,
	},
	{ // five instants with delays in between, timeframe of a single second, each new request comes so late that it is after the persistence time frame
		orderList: []order{
			{numRequests: 1, delay: 0, unit: time.Second, requestCountStart: 1, requestCountEnd: 1},
			{numRequests: 2, delay: 2, unit: time.Second, requestCountStart: 1, requestCountEnd: 2},
			{numRequests: 3, delay: 2, unit: time.Second, requestCountStart: 1, requestCountEnd: 3},
			{numRequests: 4, delay: 2, unit: time.Second, requestCountStart: 1, requestCountEnd: 4},
			{numRequests: 5, delay: 2, unit: time.Second, requestCountStart: 1, requestCountEnd: 5},
		},
		persistenceTimeframe: time.Duration(1) * time.Second,
	},
	{ // five instants with delays in between timeframe of a single second
		orderList: []order{
			{numRequests: 1000, delay: 0, unit: time.Second, requestCountStart: 1, requestCountEnd: 1000},
			{numRequests: 100, delay: 1, unit: time.Second, requestCountStart: 1001, requestCountEnd: 1100},
			{numRequests: 10, delay: 1, unit: time.Second, requestCountStart: 101, requestCountEnd: 110},
			{numRequests: 1, delay: 1, unit: time.Second, requestCountStart: 11, requestCountEnd: 11},
		},
		persistenceTimeframe: time.Duration(1) * time.Second,
	},
}

func TestHandleIndex(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal("Error creating a new request")
	}

	type appenderInfo struct {
		response   indexResponse
		orderIndex int
	}

	dispatcherToAppender := make(chan appenderInfo)
	appenderToDispatcher := make(chan bool)
	done := make(chan bool)

	//list of responses
	// Every test can have many orders
	// every order can generate many requests
	//
	// for every test, a responses list is initialized and populated with the responses of its corresponding orders
	//
	// responses -> responses of all orders of current test
	// responses[i] -> responses of order i of current
	// responses[i][j] -> response for request j of order i of current test
	//
	// to avoid memory overflows, responses will be freed after each test has completed.
	responses := make([]indexResponseList, 0, len(indexHandleTestList))

	go func() {
		for {
			appenderInfo, ok := <-dispatcherToAppender
			if ok {
				responses[appenderInfo.orderIndex] =
					append(responses[appenderInfo.orderIndex], appenderInfo.response)
				appenderToDispatcher <- true
			} else {
				break
			}
		}
	}()

	for testIndex, test := range indexHandleTestList {
		//a new server is created for each test so that each of them starts with a clean slate.
		srv := NewServer(environment{
			listenAddress:              ":5000",
			persistenceFile:            "NOT_SET",
			parsedPersistenceTimeFrame: test.persistenceTimeframe,
		})
		srv.logger.SetOutput(ioutil.Discard)
		srv.routes()
		handler := srv.index(srv.communication)

		for orderIndex, order := range test.orderList {
			time.Sleep(time.Duration(order.delay) * order.unit)

			//allocate responses array for this order index
			responses = append(responses, make(indexResponseList, 0, order.numRequests))

			for requestIndex := 0; requestIndex < order.numRequests; requestIndex++ {
				w := httptest.NewRecorder()
				go func(w *httptest.ResponseRecorder, testIndex int, orderIndex int) {

					//dispatch request to the server
					handler(w, req)
					if w.Result().StatusCode != http.StatusOK {
						t.Fatalf("Something went wrong with request '%v'\n", req)
					}

					//unpack result
					buf := new(bytes.Buffer)
					buf.ReadFrom(w.Result().Body)
					var receivedResponse indexResponse
					json.Unmarshal(buf.Bytes(), &receivedResponse)

					//send response to the appender goroutine and information where to store it
					dispatcherToAppender <- appenderInfo{
						response:   receivedResponse,
						orderIndex: orderIndex,
					}

					//wait for appender to complete
					<-appenderToDispatcher

					done <- true
				}(w, testIndex, orderIndex)
			}
		}

		//wait for as many requests as there will be sent
		for i := 0; i < test.getTotalRequests(); i++ {
			<-done
		}

		/* # Assertion analysis

		The assertion for this test is quite complex. This doc segment aims to shed some light on it.
		The intention of the assertion is to check
		- that the correct number of responses, based on the requests provided by the test, were dispatched
		- that every generated response got the right request count

		To ensure the first is straightforward:
		Per definition of the 'indexHandleTest; type, every orderList can generate many responses.
		The amount of responses generated by orderList 'ol' will correspond with 'ol.numRequests'.
		Thus, the total of expected responses is the sum of all 'oli.numRequests' for a given order list 'oli'.
		Calculating that sum:
		*/
		var numResponsesForCurrentTest int
		for _, responsesForCurrentOrder := range responses {
			//responsesForCurrentOrder is of type []responseList
			//and holds all responses for order i in responsesForCurrentOrder[i]
			//the total number of responses is, thus, the sum of lengths of these lists.
			numResponsesForCurrentTest = numResponsesForCurrentTest + len(responsesForCurrentOrder)
		}
		/*
			The assertion is made on the length of the result list:
		*/
		if numResponsesForCurrentTest != test.getTotalRequests() {
			t.Errorf("Expected to receive a total of '%v' responses for test '%v', but got '%v' instead.\n",
				test.getTotalRequests(), testIndex, numResponsesForCurrentTest)
		}
		/* The second point, checking that every generated response got the right request count, is not as trivial.

		Since each order can generate multiple responses, it is crucial for the assertion logic to establish
		a relationship between the orders in a test and the responses in the testResponses list.

		The responses associated to a specific orderList 'oli' are stored in 'responses[oli]'.
		E.g. For a test with the orderList:

			orderList: []order{
			{numRequests: 2, delay: 0, unit: time.Second, responseCountStart: 1, responseCountEnd: 2},
			{numRequests: 2, delay: 1, unit: time.Second, responseCountStart: 3, responseCountEnd: 4},
		},
		persistenceTimeframe: time.Duration(60) * time.Second,

		 The responses of the first order (index 0) will be at responses[0]
		 The responses of the second order (index 1) will at responses[1]

		Once the relationship request(test.ol) -> response(list) is established, the next piece of information
		required is which requestCount is expected for each of the responses.

		The 'persistenceTimeFrame' variable has the effect that some of the requestCounts get discarded for the
		calculations after the specified time has passed. Everything that happens within that time frame will be
		kept in the accumulated requestCount and will be reflected in the response values. After the time
		specified by the timeframe has passed, the time window moves: events outside of the time frame
		will be discarded, so the response values change. Lets consider an example:

			orderList: []order{
			{numRequests: 1, delay: 0, unit: time.Second, responseCountStart: 1, responseCountEnd: 1},
			{numRequests: 1, delay: 1, unit: time.Second, responseCountStart: 2, responseCountEnd: 2},
			{numRequests: 1, delay: 1, unit: time.Second, responseCountStart: 2, responseCountEnd: 2},
			{numRequests: 2, delay: 1, unit: time.Second, responseCountStart: 2, responseCountEnd: 3},
			{numRequests: 1, delay: 1, unit: time.Second, responseCountStart: 3, responseCountEnd: 3},
			{numRequests: 1, delay: 1, unit: time.Second, responseCountStart: 2, responseCountEnd: 2},
			{numRequests: 1, delay: 2, unit: time.Second, responseCountStart: 1, responseCountEnd: 1},
		},
		persistenceTimeframe: time.Duration(1) * time.Second,

		Let's illustrate how this persistenceTimeframe, with the value of one second, has an effect on the requests.
		There is a delay of 1 second between the first and second orders. Because the precision of the algorithm
		is of one second, these two are considered to happen within the allowed persistenceTimeframe. Consider
		that two requests come in with timestamps:

			2018-09-18 00:00:00.000000000
			2018-09-18 00:00:00.999999999

		Though the time lapse between those two is, in fact, larger than a single second, the algorithm will truncate
		it at the second units and interpret that the two timestamps occur within the allowed persistence timeFrame.

		To assert this behaviour, the test specifies that all the requests generated by a specific order must have
		a 'requestCount' varying from 'requestCountStart' and 'requestCountEnd'. In the particular case of the
		second order of the example, the single request that will be generated will take off where the previous
		request left, and have a requestCount value starting at 2 and ending at 2. This is because the previous
		request is within the persistenceTimeFrame.

		With that in mind, it follows that when the third order comes in, the request generated by the first order
		is no longer within the persistence timeframe. It's count is discarded from the totals, so now only two
		orders - the second and the third, count. Because the third order generates two requests and the requestCount
		takes off where the previous request left, their responses should get requestCounts starting at 2 and ending
		at 3.

		Note that the last request of the example specifies a delay longer that the persistence timeframe. Therefore,
		the accumulated values are reset and the expected requestCount starts back at 1.

		Combining all of these elements, the assertion follows:
		*/
		for orderIndex, order := range test.orderList {
			responsesForCurrentOrder := responses[orderIndex]
			sort.Sort(responsesForCurrentOrder)
			responseIndex := 0
			for expectedResponseCount := order.requestCountStart; expectedResponseCount <= order.requestCountEnd; expectedResponseCount++ {
				//for the current response
				currentResponse := responsesForCurrentOrder[responseIndex]
				//assert that the response has the right globalCount
				if expectedResponseCount != currentResponse.RequestCount {
					t.Fatalf("Expected received response '%v' to have a global count of '%v' but got '%v' instead. \n"+
						"Test: '%v'\n"+
						"Order: '%v'\n"+
						"Values: '%v'\n"+
						"ResponseIndex: '%v'\n",
						currentResponse, expectedResponseCount, currentResponse.RequestCount, testIndex, orderIndex, order, responseIndex)
				}
				responseIndex++
			}
		}

		//finished all assertions for this test, so I can release that space
		responses = nil
	}

	close(done)
	close(dispatcherToAppender)
	close(appenderToDispatcher)
}
