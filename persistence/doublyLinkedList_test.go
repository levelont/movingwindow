package persistence

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func dumpList(list requestCountDoublyLinkedList) string {
	var result strings.Builder
	currentNode := list.head
	for {
		if currentNode == nil {
			break
		}
		result.WriteString(strconv.Itoa(currentNode.data.RequestsCount))
		currentNode = currentNode.right
	}

	return result.String()
}

func dumpListBackwards(list requestCountDoublyLinkedList) string {
	var result strings.Builder
	currentNode := list.tail
	for {
		if currentNode == nil {
			break
		}
		result.WriteString(strconv.Itoa(currentNode.data.RequestsCount))
		currentNode = currentNode.left
	}

	return result.String()
}

func buildDoublyLinkedList(values []RequestCount) requestCountDoublyLinkedList {
	var list requestCountDoublyLinkedList
	for _, value := range values {
		list = list.AppendToTail(value)
	}

	return list
}

type appendToTailTest struct {
	listData              []RequestCount //a linked list will be constructed with these values in the same order of the slice
	expectedDump          string         //expected dumpList() output
	expectedDumpBackwards string         //expected dumpListBackwards() output
}

var appendToTailTestList = []appendToTailTest{
	{listData: []RequestCount{
		{RequestsCount: 1},
	},
		expectedDump:          "1",
		expectedDumpBackwards: "1",
	},

	{listData: []RequestCount{
		{RequestsCount: 1},
		{RequestsCount: 2},
	},
		expectedDump:          "12",
		expectedDumpBackwards: "21",
	},

	{listData: []RequestCount{
		{RequestsCount: 1},
		{RequestsCount: 2},
		{RequestsCount: 3},
		{RequestsCount: 4},
		{RequestsCount: 5},
	},
		expectedDump:          "12345",
		expectedDumpBackwards: "54321",
	},
}

func TestRequestCountDoublyLinkedList_AppendToTail(t *testing.T) {
	for i, test := range appendToTailTestList {
		list := buildDoublyLinkedList(test.listData)
		listDump := dumpList(list)
		if listDump != test.expectedDump {
			t.Fatalf("Expected '%v', got '%v' for test '%v' with values '%v'.\n", test.expectedDump, listDump, i, test)
		}
		backwardsListDump := dumpListBackwards(list)
		if backwardsListDump != test.expectedDumpBackwards {
			t.Fatalf("Expected '%v', got '%v' for test '%v' with values '%v'.\n", test.expectedDumpBackwards, backwardsListDump, i, test)
		}
	}
}

const DATE_FORMAT = time.RFC3339Nano //"2006-01-02T15:04:05.999999999Z07:00"

type withinDurationBeforeTest struct {
	reference          string
	timestamp          string
	maxAllowedDuration time.Duration // max amount of time allowed between reference and timestamp
	unit               time.Duration // unit used for the specified value of maxAllowedDuration
	expected           bool
}

var withinDurationBeforeTestList = []withinDurationBeforeTest{
	{"2006-01-02T15:04:05.000000000Z", "2006-01-02T15:04:05.000000000Z", time.Duration(0) * time.Second, time.Second, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000000000Z", time.Duration(0) * time.Second, time.Second, false},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000000000Z", time.Duration(2) * time.Second, time.Second, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000000000Z", time.Duration(1) * time.Second, time.Second, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000000001Z", time.Duration(1) * time.Second, time.Second, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.999999999Z", time.Duration(1) * time.Second, time.Second, true},
}

func TestRequestCountNode_withinDurationBefore(t *testing.T) {
	for i, test := range withinDurationBeforeTestList {
		parsedReference, err := time.Parse(DATE_FORMAT, test.reference)
		if err != nil {
			t.Fatal(err)
		}
		reference := RequestCount{Timestamp: parsedReference}

		parsedTimestamp, err := time.Parse(DATE_FORMAT, test.timestamp)
		if err != nil {
			t.Fatal(err)
		}
		data := RequestCount{Timestamp: parsedTimestamp}
		node := requestCountNode{data: data}

		result, difference := node.WithinDurationBefore(test.maxAllowedDuration, test.unit, reference)
		if result != test.expected {
			t.Fatalf("Expected '%v' but got '%v' for test '%v' with values '%v'. Difference(nanoseconds): '%v'\n", test.expected, result, i, test, difference.Nanoseconds())
		}
	}
}

type updateTotalsTest struct {
	listData     []RequestCount //a linked list will be constructed with these values in the same order of the slice
	reference    RequestCount
	expectedHead RequestCount
}

var updateTotalsTestList = []updateTotalsTest{
	{ // Head outside range
		listData: []RequestCount{
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC), RequestsCount: 1000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), RequestsCount: 1},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 51, 0, time.UTC), RequestsCount: 1},
			{Timestamp: time.Date(2006, 01, 02, 19, 01, 00, 0, time.UTC), RequestsCount: 1, accumulatedRequestCount: 1},
		},
		reference:    RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 01, 02, 0, time.UTC)},
		expectedHead: RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), RequestsCount: 1, accumulatedRequestCount: 3},
	},
	{ // Head + 3 outside range
		listData: []RequestCount{
			{Timestamp: time.Date(2006, 01, 02, 18, 59, 59, 0, time.UTC), RequestsCount: 1000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 00, 0, time.UTC), RequestsCount: 1000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC), RequestsCount: 1000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), RequestsCount: 3},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 51, 0, time.UTC), RequestsCount: 2},
			{Timestamp: time.Date(2006, 01, 02, 19, 01, 00, 0, time.UTC), RequestsCount: 1, accumulatedRequestCount: 1},
		},
		reference:    RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 01, 02, 0, time.UTC)},
		expectedHead: RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), RequestsCount: 3, accumulatedRequestCount: 6},
	},
	{ // Big values, all inside range
		listData: []RequestCount{
			{Timestamp: time.Date(2006, 01, 02, 18, 59, 59, 0, time.UTC), RequestsCount: 100000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 00, 0, time.UTC), RequestsCount: 10000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC), RequestsCount: 1000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), RequestsCount: 100},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 51, 0, time.UTC), RequestsCount: 10},
			{Timestamp: time.Date(2006, 01, 02, 19, 01, 00, 0, time.UTC), RequestsCount: 1, accumulatedRequestCount: 1},
		},
		reference:    RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 59, 0, time.UTC)},
		expectedHead: RequestCount{Timestamp: time.Date(2006, 01, 02, 18, 59, 59, 0, time.UTC), RequestsCount: 100000, accumulatedRequestCount: 111111},
	},
}

func TestRequestCountDoublyLinkedList_UpdateTotals(t *testing.T) {
	for i, test := range updateTotalsTestList {

		list := buildDoublyLinkedList(test.listData)
		list = list.UpdateTotals(test.reference)

		if list.head.data != test.expectedHead {
			t.Fatalf("Expected '%+v' but got '%+v' for test '%v' with values '%v'. List: '%v'\n", test.expectedHead.Dump(), list.head.data.Dump(), i, test, dumpList(list))
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
	firstListData  []RequestCount //a linked list will be constructed with these values in the same order of the slice
	secondListData []RequestCount //a linked list will be constructed with these values in the same order of the slice
	expected       bool
}

var compareListsTestList = []compareListsTest{
	{ // Lists are equal
		firstListData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
		secondListData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
		expected: true,
	},
	{ // Lists are empty
		firstListData:  []RequestCount{},
		secondListData: []RequestCount{},
		expected:       true,
	},
	{ // First list is empty
		firstListData: []RequestCount{},
		secondListData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
		expected: false,
	},
	{ // Second list is empty
		firstListData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
		secondListData: []RequestCount{},
		expected:       false,
	},
	{ // Lists are different at the beginning
		firstListData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
		secondListData: []RequestCount{
			{RequestsCount: 111111},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
		expected: false,
	},
	{ // Lists are different at the middle
		firstListData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
			{RequestsCount: 5},
		},
		secondListData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 333333},
			{RequestsCount: 4},
			{RequestsCount: 5},
		},
		expected: false,
	},
	{ // L1 is longer
		firstListData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
			{RequestsCount: 5},
		},
		secondListData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
		expected: false,
	},
	{ // L2 is longer
		firstListData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
		secondListData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
			{RequestsCount: 5},
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
			t.Fatalf("Expected '%v' but got '%v' from test '%v' with data '%v'\n", test.expected, result, i, test)
		}
	}
}

type frontDiscardTest struct {
	listData               []RequestCount //a linked list will be constructed with these values in the same order of the slice
	lastDataToRemove       RequestCount
	expectedResultListData []RequestCount
}

var frontDiscardTestList = []frontDiscardTest{
	{ //Discard first
		listData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
		lastDataToRemove: RequestCount{RequestsCount: 1},
		expectedResultListData: []RequestCount{
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
	},
	{ //Discard node before last
		listData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
		lastDataToRemove: RequestCount{RequestsCount: 3},
		expectedResultListData: []RequestCount{
			{RequestsCount: 4},
		},
	},
	{ //Discard last
		listData: []RequestCount{
			{RequestsCount: 1},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
		lastDataToRemove:       RequestCount{RequestsCount: 4},
		expectedResultListData: []RequestCount{},
	},
}

func findRequestCountInList(node RequestCount, list requestCountDoublyLinkedList) (*requestCountNode, error) {
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

type totalAccumulatedRequestCountTest struct {
	listData []RequestCount //a linked list will be constructed with these values in the same order of the slice
	expected int
}

var totalAccumulatedRequestCountTestList = []totalAccumulatedRequestCountTest{
	{ // Simple test
		listData: []RequestCount{
			{RequestsCount: 1, accumulatedRequestCount: 1111},
			{RequestsCount: 2},
			{RequestsCount: 3},
			{RequestsCount: 4},
		},
		expected: 1111,
	},
	{ // Never mind other values
		listData: []RequestCount{
			{RequestsCount: 1, accumulatedRequestCount: 1111},
			{RequestsCount: 2, accumulatedRequestCount: 2222},
			{RequestsCount: 3, accumulatedRequestCount: 3333},
			{RequestsCount: 4, accumulatedRequestCount: 4444},
		},
		expected: 1111,
	},
}

func TestRequestCountDoublyLinkedList_TotalAccumulatedRequestCount(t *testing.T) {
	for i, test := range totalAccumulatedRequestCountTestList {
		list := buildDoublyLinkedList(test.listData)
		result := list.TotalAccumulatedRequestCount()
		if result != test.expected {
			t.Fatalf("Expected '%v' but got '%v' from test '%v' with data '%v'\n", result, test.expected, i, test)
		}
	}
}

//requestCOunt
//timestamp
//expected
type compareTimestampWithPrecisionTest struct {
	requestCount RequestCount
	timestamp    time.Time
	precision    time.Duration //timestamp and requestCount will be compared on basis of this precision
	expected     bool
}

var compareTimestampWithPrecisionTestList = []compareTimestampWithPrecisionTest{
	{ // Equal Values
		requestCount: RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)},
		timestamp:    time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC),
		precision:    time.Second,
		expected:     true,
	},
	{ // Different values in seconds -> Different
		requestCount: RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)},
		timestamp:    time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC),
		precision:    time.Second,
		expected:     false,
	},
	{ // Different values by a single nanosecond -> Equal
		requestCount: RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)},
		timestamp:    time.Date(2006, 01, 02, 19, 00, 01, 1, time.UTC),
		precision:    time.Second,
		expected:     true,
	},
	{ // Almost equal by a single nanosecond -> Different
		requestCount: RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC)},
		timestamp:    time.Date(2006, 01, 02, 19, 00, 00, 999999999, time.UTC),
		precision:    time.Second,
		expected:     false,
	},
}

func TestRequestCount_CompareTimestampWithPrecision(t *testing.T) {
	for i, test := range compareTimestampWithPrecisionTestList {
		result := test.requestCount.CompareTimestampWithPrecision(test.timestamp, test.precision)
		if result != test.expected {
			t.Fatalf("Expected '%v' but got '%v' from test '%v' with data '%v'\n", result, test.expected, i, test)
		}
	}
}
