package persistence

import (
	"errors"
	"testing"
	"time"
)

func buildDoublyLinkedList(values []requestCount) requestCountDoublyLinkedList {
	var list requestCountDoublyLinkedList
	for _, value := range values {
		list = list.AppendToTail(value)
	}

	return list
}

type appendToTailTest struct {
	listData              []requestCount //a linked list will be constructed with these values in the same order of the slice
	expectedDump          string         //expected Dump() output
	expectedDumpBackwards string         //expected DumpBackwards() output
}

var appendToTailTestList = []appendToTailTest{
	{listData: []requestCount{
		{requestsCount: 1},
	},
		expectedDump:          "1",
		expectedDumpBackwards: "1",
	},

	{listData: []requestCount{
		{requestsCount: 1},
		{requestsCount: 2},
	},
		expectedDump:          "12",
		expectedDumpBackwards: "21",
	},

	{listData: []requestCount{
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

func TestRequestCountDoublyLinkedList_AppendToTail(t *testing.T) {
	for i, test := range appendToTailTestList {
		list := buildDoublyLinkedList(test.listData)
		if list.Dump() != test.expectedDump {
			t.Errorf("Expected '%v', got '%v' for test '%v' with values '%v'.\n", test.expectedDump, list.Dump(), i, test)
		}
		if list.DumpBackwards() != test.expectedDumpBackwards {
			t.Errorf("Expected '%v', got '%v' for test '%v' with values '%v'.\n", test.expectedDumpBackwards, list.DumpBackwards(), i, test)
		}
	}
}

const DATE_FORMAT = time.RFC3339Nano //"2006-01-02T15:04:05.999999999Z07:00"

type withinDurationFromTest struct {
	reference          string
	timestamp          string
	maxAllowedDuration time.Duration // max amount of time allowed between reference and timestamp
	unit               time.Duration // unit used for the specified value of maxAllowedDuration
	expected           bool
}

var withinDurationFromTestList = []withinDurationFromTest{
	{"2006-01-02T15:04:05.000000000Z", "2006-01-02T15:04:05.000000000Z", time.Duration(0) * time.Second, time.Second, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000000000Z", time.Duration(0) * time.Second, time.Second, false},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000000000Z", time.Duration(2) * time.Second, time.Second, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000000000Z", time.Duration(1) * time.Second, time.Second, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000000001Z", time.Duration(1) * time.Second, time.Second, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.999999999Z", time.Duration(1) * time.Second, time.Second, true},
}

func TestRequestCountNode_WithinDurationFrom(t *testing.T) {
	for i, test := range withinDurationFromTestList {
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

type updateTotalsTest struct {
	listData     []requestCount //a linked list will be constructed with these values in the same order of the slice
	reference    requestCount
	expectedHead requestCount
}

var updateTotalsTestList = []updateTotalsTest{
	{ // Head outside range
		listData: []requestCount{
			{timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC), requestsCount: 1000},
			{timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), requestsCount: 1},
			{timestamp: time.Date(2006, 01, 02, 19, 00, 51, 0, time.UTC), requestsCount: 1},
			{timestamp: time.Date(2006, 01, 02, 19, 01, 00, 0, time.UTC), requestsCount: 1, accumulatedRequestCount: 1},
		},
		reference:    requestCount{timestamp: time.Date(2006, 01, 02, 19, 01, 02, 0, time.UTC)},
		expectedHead: requestCount{timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), requestsCount: 1, accumulatedRequestCount: 3},
	},
	{ // Head + 3 outside range
		listData: []requestCount{
			{timestamp: time.Date(2006, 01, 02, 18, 59, 59, 0, time.UTC), requestsCount: 1000},
			{timestamp: time.Date(2006, 01, 02, 19, 00, 00, 0, time.UTC), requestsCount: 1000},
			{timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC), requestsCount: 1000},
			{timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), requestsCount: 3},
			{timestamp: time.Date(2006, 01, 02, 19, 00, 51, 0, time.UTC), requestsCount: 2},
			{timestamp: time.Date(2006, 01, 02, 19, 01, 00, 0, time.UTC), requestsCount: 1, accumulatedRequestCount: 1},
		},
		reference:    requestCount{timestamp: time.Date(2006, 01, 02, 19, 01, 02, 0, time.UTC)},
		expectedHead: requestCount{timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), requestsCount: 3, accumulatedRequestCount: 6},
	},
	{ // Big values, all inside range
		listData: []requestCount{
			{timestamp: time.Date(2006, 01, 02, 18, 59, 59, 0, time.UTC), requestsCount: 100000},
			{timestamp: time.Date(2006, 01, 02, 19, 00, 00, 0, time.UTC), requestsCount: 10000},
			{timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC), requestsCount: 1000},
			{timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), requestsCount: 100},
			{timestamp: time.Date(2006, 01, 02, 19, 00, 51, 0, time.UTC), requestsCount: 10},
			{timestamp: time.Date(2006, 01, 02, 19, 01, 00, 0, time.UTC), requestsCount: 1, accumulatedRequestCount: 1},
		},
		reference:    requestCount{timestamp: time.Date(2006, 01, 02, 19, 00, 59, 0, time.UTC)},
		expectedHead: requestCount{timestamp: time.Date(2006, 01, 02, 18, 59, 59, 0, time.UTC), requestsCount: 100000, accumulatedRequestCount: 111111},
	},
}

func TestRequestCountDoublyLinkedList_UpdateTotals(t *testing.T) {
	for i, test := range updateTotalsTestList {

		list := buildDoublyLinkedList(test.listData)
		list = list.UpdateTotals(test.reference)

		if list.head.data != test.expectedHead {
			t.Fatalf("Expected '%+v' but got '%+v' for test '%v' with values '%v'. Chain: '%v'\n", test.expectedHead.Dump(), list.head.data.Dump(), i, test, list.Dump())
		}
	}
}

func compareLists(l1 requestCountDoublyLinkedList, l2 requestCountDoublyLinkedList) bool {
	currentNodeL1 := l1.head
	currentNodeL2 := l2.head
	listsAreEqual := true
	for {
		if currentNodeL1 == nil {
			if currentNodeL2 != nil {
				listsAreEqual = false
			}
			break
		}

		if currentNodeL2 == nil {
			listsAreEqual = false
			break
		}

		if currentNodeL1.data != currentNodeL2.data {
			listsAreEqual = false
			break
		}

		currentNodeL1 = currentNodeL1.right
		currentNodeL2 = currentNodeL2.right
	}

	return listsAreEqual
}

type compareListsTest struct {
	firstListData  []requestCount //a linked list will be constructed with these values in the same order of the slice
	secondListData []requestCount //a linked list will be constructed with these values in the same order of the slice
	expected       bool
}

var compareListsTestList = []compareListsTest{
	{ // Lists are equal
		firstListData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
		},
		secondListData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
		},
		expected: true,
	},
	{ // Lists are empty
		firstListData:  []requestCount{},
		secondListData: []requestCount{},
		expected:       true,
	},
	{ // First list is empty
		firstListData: []requestCount{},
		secondListData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
		},
		expected: false,
	},
	{ // Second list is empty
		firstListData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
		},
		secondListData: []requestCount{},
		expected:       false,
	},
	{ // Lists are different at the beginning
		firstListData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
		},
		secondListData: []requestCount{
			{requestsCount: 111111},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
		},
		expected: false,
	},
	{ // Lists are different at the middle
		firstListData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
			{requestsCount: 5},
		},
		secondListData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 333333},
			{requestsCount: 4},
			{requestsCount: 5},
		},
		expected: false,
	},
	{ // L1 is longer
		firstListData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
			{requestsCount: 5},
		},
		secondListData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
		},
		expected: false,
	},
	{ // L2 is longer
		firstListData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
		},
		secondListData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
			{requestsCount: 5},
		},
		expected: false,
	},
}

func TestCompareLists(t *testing.T) {
	for i, test := range compareListsTestList {
		l1 := buildDoublyLinkedList(test.firstListData)
		l2 := buildDoublyLinkedList(test.secondListData)
		result := compareLists(l1, l2)
		if result != test.expected {
			t.Errorf("Expected '%v' but got '%v' from test '%v' with data '%v'\n", test.expected, result, i, test)
		}
	}
}

type frontDiscardTest struct {
	listData               []requestCount //a linked list will be constructed with these values in the same order of the slice
	lastDataToRemove       requestCount
	expectedResultListData []requestCount
}

var frontDiscardTestList = []frontDiscardTest{
	{ //Discard first
		listData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
		},
		lastDataToRemove: requestCount{requestsCount: 1},
		expectedResultListData: []requestCount{
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
		},
	},
	{ //Discard node before last
		listData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
		},
		lastDataToRemove: requestCount{requestsCount: 3},
		expectedResultListData: []requestCount{
			{requestsCount: 4},
		},
	},
	{ //Discard last
		listData: []requestCount{
			{requestsCount: 1},
			{requestsCount: 2},
			{requestsCount: 3},
			{requestsCount: 4},
		},
		lastDataToRemove:       requestCount{requestsCount: 4},
		expectedResultListData: []requestCount{},
	},
}

func findRequestCountInList(node requestCount, list requestCountDoublyLinkedList) (*requestCountNode, error) {
	currentNode := list.head
	var result *requestCountNode
	for {
		if currentNode == nil {
			break
		}

		if currentNode.data == node {
			result = currentNode
			break
		}

		currentNode = currentNode.right
	}

	if result == nil {
		return nil, errors.New("Node not found")
	}

	return result, nil
}

func TestRequestCountDoublyLinkedList_FrontDiscardUntil(t *testing.T) {
	for i, test := range frontDiscardTestList {
		initialList := buildDoublyLinkedList(test.listData)
		lastNodeToDiscard, err := findRequestCountInList(test.lastDataToRemove, initialList)
		if err != nil {
			t.Fatal(err)
		}
		expectedListAfterDiscard := buildDoublyLinkedList(test.expectedResultListData)
		resultList := initialList.FrontDiscardUntil(lastNodeToDiscard)
		if !compareLists(expectedListAfterDiscard, resultList) {
			t.Fatalf("Expected '%v' but got '%v' from test '%v' with data '%v'\n", expectedListAfterDiscard, resultList, i, test)
		}
	}
}
