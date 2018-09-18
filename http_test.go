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

//use an array to store requestCounts
//the result of the request is a marshalled JSON that we can unmarshall into a requestCount object
//we just need the structure to implement sort interface with
//TODO move these variables and the test somewhere else? or just document properly.
type responseList []persistence.Cache

func (s responseList) Len() int {
	return len(s)
}
func (s responseList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s responseList) Less(i, j int) bool {
	if s[i].CompareTimestampWithPrecision(s[j].Timestamp, time.Second) {
		return s[i].RequestsCount < s[j].RequestsCount
	} else {
		return s[i].Timestamp.Before(s[j].Timestamp)
	}
}

type requestCountListSortingTest struct {
	unsorted responseList
	sorted   responseList
}

var requestCountListSortingTestList = []requestCountListSortingTest{
	{ // Basic test
		unsorted: responseList{
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
		},
		sorted: responseList{
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
		},
	},

	{ // Timestamp collisions
		unsorted: responseList{
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 3}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 1}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
		},
		sorted: responseList{
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 1}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 3}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
		},
	},
	{ // Timestamp and request count collisions
		unsorted: responseList{
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 3}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 1}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
		},
		sorted: responseList{
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 1}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 2}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 03, 0, time.UTC), RequestsCount: 3}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 04, 0, time.UTC)}},
			persistence.Cache{RequestCount: persistence.RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 05, 0, time.UTC)}},
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

//parms for index testing
//'intervals' for requests and ammount of requests to be sent for each of em
//total requests at the end

//TODO better naming: order and orderlist
type order struct {
	numRequests   int
	delay         int
	unit          time.Duration
	expectedTotal int
}

type indexHandleTest struct {
	orderList            []order
	persistenceTimeframe time.Duration
}

func (i indexHandleTest) getTotalRequests() int {
	totalRequests := 0
	for _, order := range i.orderList {
		totalRequests = totalRequests + order.numRequests
	}
	return totalRequests
}

//TODO tests temporarily out-commented to improve turnaround speed.
//TODO re-enable
//TODO still, tests are taking 5s. Why? Is logging degrading performance although it is not writing to a proper device?
var indexHandleTestList = []indexHandleTest{
	{ //single instant, 1 request, 'infinite' persistenceTimeframe
		orderList: []order{
			{numRequests: 1, delay: 0, unit: time.Second, expectedTotal: 1},
		},
		persistenceTimeframe: time.Duration(60) * time.Second,
	},
	{ //two instants, 1s=2 2s=2, 'infinite' persistenceTimeFrame
		orderList: []order{
			{numRequests: 2, delay: 0, unit: time.Second, expectedTotal: 2},
			{numRequests: 2, delay: 1, unit: time.Second, expectedTotal: 4},
		},
		persistenceTimeframe: time.Duration(60) * time.Second,
	},
	/*{ //single instant, 1000 requests, 'infinite' persistenceTimeFrame
		orderList: []order{
			{numRequests: 1000, delay: 0, unit: time.Second, expectedTotal: 1000},
		},
		persistenceTimeframe: time.Duration(60) * time.Second,
	},*/
	/*{ // three consecutive instants, 1s=1000, 2s=1000, 3s=1000, 'infinite' persistenceTimeFrame
		orderList: []order{
			{numRequests: 1000, delay: 0, unit: time.Second, expectedTotal: 1000},
			{numRequests: 1000, delay: 1, unit: time.Second, expectedTotal: 2000},
			{numRequests: 1000, delay: 1, unit: time.Second, expectedTotal: 3000},
		},
		persistenceTimeframe: time.Duration(60) * time.Second,
	},*/
	{ // five instants with delays in between, timeframe of a single second
		orderList: []order{
			{numRequests: 1, delay: 0, unit: time.Second, expectedTotal: 1},
			{numRequests: 1, delay: 1, unit: time.Second, expectedTotal: 2},
			{numRequests: 1, delay: 1, unit: time.Second, expectedTotal: 2},
			{numRequests: 1, delay: 1, unit: time.Second, expectedTotal: 2},
			{numRequests: 1, delay: 1, unit: time.Second, expectedTotal: 2},
		},
		persistenceTimeframe: time.Duration(1) * time.Second,
	},
	/*{ // five instants with delays in between timeframe of a single second
		orderList: []order{
			{numRequests: 1000, delay: 1, unit: time.Second, expectedTotal: 1000},
			{numRequests: 100, delay: 1, unit: time.Second, expectedTotal: 1100},
			{numRequests: 10, delay: 1, unit: time.Second, expectedTotal: 110},
			{numRequests: 1, delay: 1, unit: time.Second, expectedTotal: 11},
		},
		persistenceTimeframe: time.Duration(1) * time.Second,
	},*/
}

func TestHandleIndex(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal("Error creating a new request")
	}

	type responseTestIndexTuple struct {
		response  persistence.Cache
		testIndex int
	}

	dispatcherToAppender := make(chan responseTestIndexTuple)
	appenderToDispatcher := make(chan bool)
	done := make(chan bool)

	//list of responseLists.
	// Every test has its own list of responses.
	// Responses of test i can be located at responses[i]
	responses := make([]responseList, 0, len(indexHandleTestList))

	go func() {
		for {
			log.Println("APPENDER: Waiting for response...")
			responseTestIndexTuple, ok := <-dispatcherToAppender
			if ok {
				log.Printf("APPENDER: Accessing possition '%v' of responses '%v'\n", responseTestIndexTuple.testIndex, responses)
				responses[responseTestIndexTuple.testIndex] = append(responses[responseTestIndexTuple.testIndex], responseTestIndexTuple.response)
				log.Printf("APPENDER: Successfully added response '%v' to results of test '%v'\n", responseTestIndexTuple.response, responseTestIndexTuple.testIndex)
				appenderToDispatcher <- true
			} else {
				break
			}
		}
	}()

	//TODO names
	//TODO hideous indentation
	var w *httptest.ResponseRecorder
	for testIndex, test := range indexHandleTestList {
		log.Printf("Test '%v' - values: '%v'\n", testIndex, test)
		srv := NewServer(":5000", test.persistenceTimeframe)
		srv.logger.SetOutput(ioutil.Discard)
		srv.routes()
		handler := srv.index(srv.communication)
		responses = append(responses, make(responseList, 0, test.getTotalRequests()))

		for _, order := range test.orderList {
			time.Sleep(time.Duration(order.delay) * order.unit)

			for i := 0; i < order.numRequests; i++ { // individual num requests of each test's order
				w = httptest.NewRecorder()

				go func(w *httptest.ResponseRecorder) {
					handler(w, req)

					if w.Result().StatusCode != http.StatusOK {
						t.Fatalf("Something went wrong with request '%v'\n", req)
					}

					buf := new(bytes.Buffer)
					buf.ReadFrom(w.Result().Body)

					var receivedResponse persistence.Cache
					json.Unmarshal(buf.Bytes(), &receivedResponse)
					log.Printf("DISPATCHER: Received response: '%v'\n", receivedResponse)

					dispatcherToAppender <- responseTestIndexTuple{response: receivedResponse, testIndex: testIndex}
					<-appenderToDispatcher
					log.Print("DISPATCHER: Finished processing.")
					done <- true
				}(w)
			}
		}

		//need to wait for as many requests as there will be sent, not only for the valid ones
		for i := 0; i < test.getTotalRequests(); i++ {
			<-done
		}

		testResponses := responses[testIndex]

		/* # Assertion analysis

		The assertion for this test is quite complex. This doc segment aims to shed some light on it.
		The intention of the assertion is to check
		- that the correct number of responses, based on the requests provided by the test, were dispatched
		- that every generated response got the right result

		To ensure the first is straightforward:
		Per definition of the 'indexHandleTest; type, every orderList can generate many responses.
		The amount of responses generated by orderList ol will correspond with ol.numRequests.
		Thus, the total of expected responses is the sum of all oli.numRequests for a given order list oli.
		The assertion is made on the length of the result list:
		*/
		if len(testResponses) != test.getTotalRequests() {
			t.Errorf("Expected to receive '%v' response for test '%v', but got '%v' instead.\n",
				test.getTotalRequests(), testIndex, len(testResponses))
		}
		/* The second point, checking that every generated response got the right value, is not as trivial.

		Since each order can generate multiple responses, it is crucial for the assertion logic to establish
		a relationship between the orders in a test and the responses in the testResponses list.

		In order to locate the responses associated to a specific orderList in the 'responses' multi-array, the follow applies:

			ResponsesOfOrderList(oli) : responses [oli-1.numRequests, oli-1.numRequests + oli.numRequests)
			| oli-1 < 0 ? -> oli-1.numRequests = 0

		 Note that the interval is open on the left side; that is, the right limit is not-inclusive.

		 E.g. For a test with the orderList:

			orderList: []order{
				{numRequests: 1000, delay: 1, unit: time.Second, expectedTotal: 1000},
				{numRequests: 100, delay: 1, unit: time.Second, expectedTotal: 1100},
				{numRequests: 10, delay: 1, unit: time.Second, expectedTotal: 110},
				{numRequests: 1, delay: 1, unit: time.Second, expectedTotal: 11},
			}

		 The responses of the first order will start at 0 and finish at 999
		 The responses of the second order will start at 1000 and finish at 1099
		 The responses of the third order will start at 1100 and finish at 1109
		 The responses of the fourth order will start at 1110 and finish at 1110 // only one item

		Once the relationship request(test.ol) -> response(list) is established, the next piece of information
		required is which requestCount is expected for each of the responses.

		In the basic case, where the 'persistenceTimeFrame' is infinite, if n responses were generated, each
		response ri should get a requestCount 1 <= rc <= n.
		In this case, establishing that a correct response was generated for each request would be satisfied by
		simply ensuring that a total of n responses were generated and that the requestCount values of them span
		the entire integer interval of [1, n], discretely, without any holes. This can be asserted by sorting
		the list of responses and ensuring that, in the sorted list, response ri with requestCount 1 <= i <= n
		is located at position i of the list.
		Sorting is, therefore, the next step of the algorithm.
		*/
		sort.Sort(testResponses)
		/*There is, however, a non-trivial case, in which the 'persistenceTimeFrame' variable has the effect that
		some of the requestCounts get discarded after the specified time has passed. Everything that happens within
		a that time frame will be kept in the accumulated requestCount and will be reflected in the response values.
		After the time specified by the timeframe has passed, the time window moves: events outside of the time frame
		will be discarded, so so the response values change.
		Going back to the example:

			orderList: []order{
				{numRequests: 1, delay: 0, unit: time.Second, expectedTotal: 1},
				{numRequests: 1, delay: 1, unit: time.Second, expectedTotal: 2},
				{numRequests: 1, delay: 1, unit: time.Second, expectedTotal: 2},
				{numRequests: 1, delay: 1, unit: time.Second, expectedTotal: 2},
				{numRequests: 1, delay: 1, unit: time.Second, expectedTotal: 2},
			},
			persistenceTimeframe: time.Duration(1) * time.Second,

		Let's illustrate how this persistenceTimeframe, with the value of one second, has an effect on the requests.
		There is a delay of 1 second between the first and second orders. Because the precision of the algorithm
		is of one second, these two are considered to happen within the allowed persistenceTimeframe. To illustrate
		this: consider that two requests come in with timestamps:

			2018-09-18 00:00:00.000000000
			2018-09-18 00:00:00.999999999

		Though the time lapse between those two is, in fact, larger than a single second, the algorithm will truncate
		it at the second units and interpret that the two timestamps occur within the allowed persistenceTimeFrame.
		With that in mind, it follows that when the third order comes in, the request generated by the first order
		is no longer within the persistenceTimeframe. It's count is discarded from the totals, so now only two
		orders - the second and the third, count. That is the meaning of 'expectedTotal' being set to 2 in the
		third order.

		Now, the example above is simple in that each order only gives place to a single request, ergo to a single
		response.
		When the non-trivial case generates more than one response per order, there is an additional variable
		to take into account. For example, with the orders:

			orderList: []order{
				{numRequests: 1000, delay: 1, unit: time.Second, expectedTotal: 1000},
				{numRequests: 100, delay: 1, unit: time.Second, expectedTotal: 1100},
				{numRequests: 10, delay: 1, unit: time.Second, expectedTotal: 110},
				{numRequests: 1, delay: 1, unit: time.Second, expectedTotal: 11},
			},
			persistenceTimeframe: time.Duration(1) * time.Second,

		The first order will generate a thousand requests. Since these requests will be handled concurrently,
		they will all fit in the persistence timeframe of one second. Each of them should get the right requestCount
		in their responses. For example, the first request gets a count of 1, the second a count of 2, the last one
		a count of 1000.
		Note how the values range from 1 to that specified by the expectedTotal member of the current order.
		Analogous to the example before this one, the responses of the second order will also fit within the
		persistenceTimeframe. Thus, the requestsCount for these responses will start from where they left of. That
		is, the first response will get a requestCount of 1001, the second a count of 1002, all the way til the
		last one with a count of 1100.
		Note how the values started where they left of, and go all the wat to the expectedTotal member of the
		current order.
		In general, thus, the following applies

			RequestCountOfOrder(oli) -> [oli-1.numRequests, oli.expectedTotal]
			| oli-1 < 0 ? -> oli-1.numRequests = 1

		Combining all of these elements, the assertion follows:
		*/
		lastNumRequests := 0
		for _, order := range test.orderList {
			for responseIndex := lastNumRequests; responseIndex < lastNumRequests+order.numRequests; responseIndex++ {
				response := testResponses[responseIndex]
				expectedTotalForCurrentResponse := responseIndex + 1
				if expectedTotalForCurrentResponse != response.AccumulatedRequestCount {
					t.Errorf("Expected received response '%v' to have a count of '%v' but got '%v' instead. Test; '%v', Values: '%v'\n",
						response, expectedTotalForCurrentResponse, response.AccumulatedRequestCount, testIndex, test)
				}
			}
			lastNumRequests = order.numRequests
		}
	}

	close(done)
	close(dispatcherToAppender)
	close(appenderToDispatcher)
}
