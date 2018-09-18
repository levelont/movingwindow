package persistence

import (
	"os"
	"reflect"
	"testing"
	"time"
)

type encodeStateTest struct {
	statePastData requestCountList //a linked list will be constructed with these values in the same order of the slice
	statePresent  Cache
}

var encodeStateTestList = []encodeStateTest{
	{ // varied values
		statePastData: requestCountList{
			{Timestamp: time.Date(1111, 11, 11, 11, 11, 11, 111111111, time.UTC), Count: 1, Accumulated: 11},
			{Timestamp: time.Date(2222, 22, 22, 22, 22, 22, 222222222, time.UTC), Count: 2, Accumulated: 22},
			{Timestamp: time.Date(3333, 33, 33, 33, 33, 33, 333333333, time.UTC), Count: 3, Accumulated: 33},
			{Timestamp: time.Date(4444, 44, 44, 44, 44, 44, 444444444, time.UTC), Count: 4, Accumulated: 44},
		},
		statePresent: Cache{
			RequestCount:                 RequestCount{Timestamp: time.Date(5555, 55, 55, 55, 55, 55, 555555555, time.UTC), Count: 5, Accumulated: 55},
			TotalRequestsWithinTimeframe: 6666,
		},
	},
	{ // no values
		statePastData: requestCountList{},
		statePresent:  Cache{},
	},
}

//encode and file is not present
//encode and file is present -> overwrite
//read and no file is present -> error
//read and path is error -> error

func TestEncodeState(t *testing.T) {
	//setup
	testDir := "test_data"
	err := os.MkdirAll(testDir, 0777)
	if err != nil {
		t.Fatalf("Error creating test data directory during test setup: '%v'\n", err)
	}
	filePath := testDir + "/encodedState.bin"

	for testIndex, test := range encodeStateTestList {
		providedState := State{Past: test.statePastData.BuildDoublyLinkedList(), Present: test.statePresent}
		err := providedState.WriteToFile(filePath)
		if err != nil {
			t.Fatalf("Error writing state to path '%v'.\nTest: '%v'\n Data: '%v'\n \nError: '%v'\n", filePath, testIndex, test, err)
		}

		stateReadFromFile, err := ReadFromFile(filePath)
		if err != nil {
			t.Fatalf("Error reading state from file '%v'. Test: '%v'\n Data: '%v'\n Error: '%v'", filePath, testIndex, test, err)
		}

		/* # Assertion analysis

		The 'past' member of a State type is a doublyLinkedList, merely a pair of memory addresses.
		A deep comparison will compare the referenced objects, so comparing on this level is not a mistake
		*/
		if !reflect.DeepEqual(providedState, stateReadFromFile) {
			t.Fatalf("Expected state read from file '%v' to match the original, but didn't.\n Test '%v'\n Data '%v'\n Expected: '%v'\n Result: '%v'\n", filePath, testIndex, test, providedState, stateReadFromFile)
		}
		/* However, that is not enough, as that does not take into account the intermediate nodes between the
		head and the tail of the doublyLinkedList. To do that, we need to compare the nodes of the provided and
		read doublyLinkedListFiles
		*/
		providedStateNodes := providedState.Past.getNodes()
		stateReadFromFileNodes := stateReadFromFile.Past.getNodes()
		if !reflect.DeepEqual(providedStateNodes, stateReadFromFileNodes) {
			t.Fatalf("Expected internal state read from file '%v' to match the original, but didn't."+
				"\n Test '%v'\n "+
				"Data '%v'\n "+
				"Expected: '%v'\n "+
				"Result: '%v'\n",
				filePath, testIndex, test, providedStateNodes, stateReadFromFileNodes)
		}
	}

	//teardown
	err = os.RemoveAll(testDir)
	if err != nil {
		t.Fatalf("Error removing test data directory during test tear down: '%v'\n", err)
	}
}
