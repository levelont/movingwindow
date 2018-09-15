package persistence

import (
	"testing"
	"time"
)

func TestRequestCountDoublyLinkedList_AppendToTail(t *testing.T) {
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
