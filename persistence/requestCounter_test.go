package persistence

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func dumpList(list RequestCounter) string {
	var result strings.Builder
	currentNode := list.head
	for {
		if currentNode == nil {
			break
		}
		result.WriteString(strconv.Itoa(currentNode.data.Count))
		currentNode = currentNode.right
	}

	return result.String()
}

func dumpListBackwards(list RequestCounter) string {
	var result strings.Builder
	currentNode := list.tail
	for {
		if currentNode == nil {
			break
		}
		result.WriteString(strconv.Itoa(currentNode.data.Count))
		currentNode = currentNode.left
	}

	return result.String()
}

type appendToTailTest struct {
	listData              requestCountList //a linked list will be constructed with these values in the same order of the slice
	expectedDump          string           //expected dumpList() output
	expectedDumpBackwards string           //expected dumpListBackwards() output
}

var appendToTailTestList = []appendToTailTest{
	{listData: requestCountList{
		{Count: 1},
	},
		expectedDump:          "1",
		expectedDumpBackwards: "1",
	},

	{listData: requestCountList{
		{Count: 1},
		{Count: 2},
	},
		expectedDump:          "12",
		expectedDumpBackwards: "21",
	},

	{listData: requestCountList{
		{Count: 1},
		{Count: 2},
		{Count: 3},
		{Count: 4},
		{Count: 5},
	},
		expectedDump:          "12345",
		expectedDumpBackwards: "54321",
	},
}

func TestRequestCounter_AppendToTail(t *testing.T) {
	for i, test := range appendToTailTestList {
		list := test.listData.ToRequestCounter()
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

	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.999999997Z", time.Duration(2) * time.Nanosecond, time.Nanosecond, false},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.999999998Z", time.Duration(2) * time.Nanosecond, time.Nanosecond, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.999999998Z", time.Duration(1) * time.Nanosecond, time.Nanosecond, false},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.999999999Z", time.Duration(1) * time.Nanosecond, time.Nanosecond, true},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.000999999Z", time.Duration(1) * time.Millisecond, time.Millisecond, false},
	{"2006-01-02T15:04:06.000000000Z", "2006-01-02T15:04:05.001000000Z", time.Duration(1) * time.Millisecond, time.Millisecond, false},
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
	listData     requestCountList //a linked list will be constructed with these values in the same order of the slice
	reference    RequestCount
	timeframe    time.Duration
	precision    time.Duration
	expectedHead RequestCount
}

var updateTotalsTestList = []updateTotalsTest{
	{ // Head outside range
		listData: requestCountList{
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC), Count: 1000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), Count: 1},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 51, 0, time.UTC), Count: 1},
			{Timestamp: time.Date(2006, 01, 02, 19, 01, 00, 0, time.UTC), Count: 1, Accumulated: 1},
		},
		reference:    RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 01, 02, 0, time.UTC)},
		timeframe:    time.Duration(60) * time.Second,
		precision:    time.Second,
		expectedHead: RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), Count: 1, Accumulated: 3},
	},
	{ // Head + 3 outside range
		listData: requestCountList{
			{Timestamp: time.Date(2006, 01, 02, 18, 59, 59, 0, time.UTC), Count: 1000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 00, 0, time.UTC), Count: 1000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC), Count: 1000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), Count: 3},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 51, 0, time.UTC), Count: 2},
			{Timestamp: time.Date(2006, 01, 02, 19, 01, 00, 0, time.UTC), Count: 1, Accumulated: 1},
		},
		reference:    RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 01, 02, 0, time.UTC)},
		timeframe:    time.Duration(60) * time.Second,
		precision:    time.Second,
		expectedHead: RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), Count: 3, Accumulated: 6},
	},
	{ // Big values, all inside range but the first one
		listData: requestCountList{
			{Timestamp: time.Date(2006, 01, 02, 18, 59, 58, 0, time.UTC), Count: 999999},
			{Timestamp: time.Date(2006, 01, 02, 18, 59, 59, 0, time.UTC), Count: 100000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 00, 0, time.UTC), Count: 10000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC), Count: 1000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), Count: 100},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 51, 0, time.UTC), Count: 10},
			{Timestamp: time.Date(2006, 01, 02, 19, 01, 00, 0, time.UTC), Count: 1, Accumulated: 1},
		},
		reference:    RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 59, 0, time.UTC)},
		timeframe:    time.Duration(60) * time.Second,
		precision:    time.Second,
		expectedHead: RequestCount{Timestamp: time.Date(2006, 01, 02, 18, 59, 59, 0, time.UTC), Count: 100000, Accumulated: 111111},
	},
	{ // Millisecond precision, Head is outside Range by a single millisecond
		listData: requestCountList{
			//OUT
			//{Timestamp: time.Date(2006, 01, 02, 18, 59, 58, 1000000, time.UTC), Count: 100000},
			//{Timestamp: time.Date(2006, 01, 02, 18, 59, 58, 000100000, time.UTC), Count: 100000},
			// IN {Timestamp: time.Date(2006, 01, 02, 18, 59, 58, 999099999, time.UTC), Count: 100000},
			{Timestamp: time.Date(2006, 01, 02, 18, 59, 58, 999000000, time.UTC), Count: 999999},
			{Timestamp: time.Date(2006, 01, 02, 18, 59, 58, 999000001, time.UTC), Count: 100000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 00, 0, time.UTC), Count: 10000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 01, 0, time.UTC), Count: 1000},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 02, 0, time.UTC), Count: 100},
			{Timestamp: time.Date(2006, 01, 02, 19, 00, 51, 0, time.UTC), Count: 10},
			{Timestamp: time.Date(2006, 01, 02, 19, 01, 00, 0, time.UTC), Count: 1, Accumulated: 1},
		},
		reference:    RequestCount{Timestamp: time.Date(2006, 01, 02, 19, 00, 59, 0, time.UTC)},
		timeframe:    time.Duration(60) * time.Second,
		precision:    time.Millisecond,
		expectedHead: RequestCount{Timestamp: time.Date(2006, 01, 02, 18, 59, 58, 999000001, time.UTC), Count: 100000, Accumulated: 111111},
	},
}

func TestRequestCounter_UpdateTotals(t *testing.T) {
	for i, test := range updateTotalsTestList {
		list := test.listData.ToRequestCounter()
		list = list.UpdateTotals(test.reference, test.timeframe, test.precision)

		if list.head.data != test.expectedHead {
			t.Fatalf("Expected '%+v' but got '%+v' for test '%v' with values '%v'. List: '%v'\n", test.expectedHead.Dump(), list.head.data.Dump(), i, test, dumpList(list))
		}
	}
}

func compareLists(l1 RequestCounter, l2 RequestCounter) bool {
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
	firstListData  requestCountList //a linked list will be constructed with these values in the same order of the slice
	secondListData requestCountList //a linked list will be constructed with these values in the same order of the slice
	expected       bool
}

var compareListsTestList = []compareListsTest{
	{ // Lists are equal
		firstListData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
		secondListData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
		expected: true,
	},
	{ // Lists are empty
		firstListData:  requestCountList{},
		secondListData: requestCountList{},
		expected:       true,
	},
	{ // First list is empty
		firstListData: requestCountList{},
		secondListData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
		expected: false,
	},
	{ // Second list is empty
		firstListData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
		secondListData: requestCountList{},
		expected:       false,
	},
	{ // Lists are different at the beginning
		firstListData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
		secondListData: requestCountList{
			{Count: 111111},
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
		expected: false,
	},
	{ // Lists are different at the middle
		firstListData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
			{Count: 5},
		},
		secondListData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 333333},
			{Count: 4},
			{Count: 5},
		},
		expected: false,
	},
	{ // L1 is longer
		firstListData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
			{Count: 5},
		},
		secondListData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
		expected: false,
	},
	{ // L2 is longer
		firstListData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
		secondListData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
			{Count: 5},
		},
		expected: false,
	},
}

func TestCompareLists(t *testing.T) {
	for i, test := range compareListsTestList {
		l1 := test.firstListData.ToRequestCounter()
		l2 := test.secondListData.ToRequestCounter()
		result := compareLists(l1, l2)
		if result != test.expected {
			t.Fatalf("Expected '%v' but got '%v' from test '%v' with data '%v'\n", test.expected, result, i, test)
		}
	}
}

type frontDiscardTest struct {
	listData               requestCountList //a linked list will be constructed with these values in the same order of the slice
	lastDataToRemove       RequestCount
	expectedResultListData requestCountList
}

var frontDiscardTestList = []frontDiscardTest{
	{ //Discard first
		listData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
		lastDataToRemove: RequestCount{Count: 1},
		expectedResultListData: requestCountList{
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
	},
	{ //Discard node before last
		listData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
		lastDataToRemove: RequestCount{Count: 3},
		expectedResultListData: requestCountList{
			{Count: 4},
		},
	},
	{ //Discard last
		listData: requestCountList{
			{Count: 1},
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
		lastDataToRemove:       RequestCount{Count: 4},
		expectedResultListData: requestCountList{},
	},
}

func findRequestCountInList(node RequestCount, list RequestCounter) (*requestCountNode, error) {
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

func TestRequestCounter_FrontDiscardUntil(t *testing.T) {
	for i, test := range frontDiscardTestList {
		initialList := test.listData.ToRequestCounter()
		lastNodeToDiscard, err := findRequestCountInList(test.lastDataToRemove, initialList)
		if err != nil {
			t.Fatal(err)
		}
		expectedListAfterDiscard := test.expectedResultListData.ToRequestCounter()
		resultList := initialList.frontDiscardUntil(lastNodeToDiscard)
		if !compareLists(expectedListAfterDiscard, resultList) {
			t.Fatalf("Expected '%v' but got '%v' from test '%v' with data '%v'\n", expectedListAfterDiscard, resultList, i, test)
		}
	}
}

type totalAccumulatedRequestCountTest struct {
	listData requestCountList //a linked list will be constructed with these values in the same order of the slice
	expected int
}

var totalAccumulatedRequestCountTestList = []totalAccumulatedRequestCountTest{
	{ // Simple test
		listData: requestCountList{
			{Count: 1, Accumulated: 1111},
			{Count: 2},
			{Count: 3},
			{Count: 4},
		},
		expected: 1111,
	},
	{ // Never mind other values
		listData: requestCountList{
			{Count: 1, Accumulated: 1111},
			{Count: 2, Accumulated: 2222},
			{Count: 3, Accumulated: 3333},
			{Count: 4, Accumulated: 4444},
		},
		expected: 1111,
	},
}

func TestRequestCounter_TotalAccumulatedRequestCount(t *testing.T) {
	for i, test := range totalAccumulatedRequestCountTestList {
		list := test.listData.ToRequestCounter()
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
