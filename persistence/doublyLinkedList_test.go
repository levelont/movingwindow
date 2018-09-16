package persistence

import (
	"testing"
	"time"
)

type appendToTailTest struct {
	values                []requestCount
	expectedDump          string
	expectedDumpBackwards string
}

var appendToTailTestList = []appendToTailTest{
	{values: []requestCount{
		{requestsCount: 1},
	},
		expectedDump:          "1",
		expectedDumpBackwards: "1",
	},

	{values: []requestCount{
		{requestsCount: 1},
		{requestsCount: 2},
	},
		expectedDump:          "12",
		expectedDumpBackwards: "21",
	},

	{values: []requestCount{
		{requestsCount: 1},
		{requestsCount: 2},
		{requestsCount: 3},
		{requestsCount: 4},
		{requestsCount: 5},
	},
		expectedDump:          "12345",
		expectedDumpBackwards: "54321",
	},
}

func buildDoublyLinkedList(values []requestCount) requestCountDoublyLinkedList {
	var list requestCountDoublyLinkedList
	for _, value := range values {
		list = list.AppendToTail(value)
	}

	return list
}

func TestRequestCountDoublyLinkedList_AppendToTail(t *testing.T) {
	for i, test := range appendToTailTestList {
		list := buildDoublyLinkedList(test.values)
		if list.Dump() != test.expectedDump {
			t.Errorf("Expected '%v', got '%v' for test '%v' with values '%v'.\n", test.expectedDump, list.Dump(), i, test)
		}
		if list.DumpBackwards() != test.expectedDumpBackwards {
			t.Errorf("Expected '%v', got '%v' for test '%v' with values '%v'.\n", test.expectedDumpBackwards, list.DumpBackwards(), i, test)
		}
	}

	list := requestCountDoublyLinkedList{}
	list = list.AppendToTail(requestCount{requestsCount: 1})
	list = list.AppendToTail(requestCount{requestsCount: 2})
	expected := "12"
	if list.Dump() != expected {
		t.Errorf("Expected '%v', got '%v'\n", expected, list.Dump())
	}
}

//const DATE_FORMAT = "2006-01-02 15:04:05"
const DATE_FORMAT = time.RFC3339Nano //"2006-01-02T15:04:05.999999999Z07:00"

// A maxAllowedDuration will be allowed between reference and timestamp
type durationTest struct {
	reference          string
	timestamp          string
	maxAllowedDuration time.Duration
	unit               time.Duration
	expected           bool
}

var durationTestList = []durationTest{
	{"2006-01-02T15:04:05.000000000Z", "2006-01-02T15:04:05.000000000Z", time.Duration(0) * time.Second, time.Second, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000000000Z", time.Duration(0) * time.Second, time.Second, false},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000000000Z", time.Duration(2) * time.Second, time.Second, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000000000Z", time.Duration(1) * time.Second, time.Second, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000000001Z", time.Duration(1) * time.Second, time.Second, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.999999999Z", time.Duration(1) * time.Second, time.Second, true},
}

func TestRequestCountNode_WithinDurationFrom(t *testing.T) {
	for i, test := range durationTestList {
		parsedReference, err := time.Parse(DATE_FORMAT, test.reference)
		if err != nil {
			t.Fatal(err)
		}
		reference := requestCount{timestamp: parsedReference}

		parsedTimestamp, err := time.Parse(DATE_FORMAT, test.timestamp)
		if err != nil {
			t.Fatal(err)
		}
		data := requestCount{timestamp: parsedTimestamp}
		node := requestCountNode{data: data}

		result, difference := node.WithinDurationFrom(test.maxAllowedDuration, test.unit, reference)
		if result != test.expected {
			t.Fatalf("Expected '%v' but got '%v' for test '%v' with values '%v'. Difference(nanoseconds): '%v'\n", test.expected, result, i, test, difference.Nanoseconds())
		}
	}
}

//need a chain
//a reference
//an expected head with a total, timestamp and total count so far

type updateTotalsTest struct {
	values       []requestCount
	reference    requestCount
	expectedHead requestCount
}

var updateTotalsTestList = []updateTotalsTest{
	{values: []requestCount{
		{timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC), requestsCount: 2, accumulatedRequestCount: 2},
		{timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), requestsCount: 1, accumulatedRequestCount: 1},
		{timestamp: time.Date(2006, 01, 02, 19, 00, 51, 0, time.UTC), requestsCount: 1, accumulatedRequestCount: 1},
		{timestamp: time.Date(2006, 01, 02, 19, 01, 00, 0, time.UTC), requestsCount: 1, accumulatedRequestCount: 1},
	},
		reference:    requestCount{timestamp: time.Date(2006, 01, 02, 19, 01, 02, 0, time.UTC), requestsCount: 1},
		expectedHead: requestCount{timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), requestsCount: 1, accumulatedRequestCount: 3},
	},
}

func TestRequestCountDoublyLinkedList_UpdateTotals(t *testing.T) {
	for i, test := range updateTotalsTestList {

		list := buildDoublyLinkedList(test.values)
		list = list.UpdateTotals(test.reference)

		if list.head.data != test.expectedHead {
			t.Fatalf("Expected '%+v' but got '%+v' for test '%v' with values '%v'. Chain: '%v'\n", test.expectedHead.Dump(), list.head.data.Dump(), i, test, list.Dump())
		}
	}
}
