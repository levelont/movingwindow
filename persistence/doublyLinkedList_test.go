package persistence

import "testing"

func TestAdd(t *testing.T) {
	list := requestCountDoublyLinkedList{}
	list = list.AppendToTail(requestCount{requestsCount: 1})
	list = list.AppendToTail(requestCount{requestsCount: 2})
	expected := "12"
	if list.Dump() != expected {
		t.Errorf("Expected '%v', got '%v'\n", expected, list.Dump())
	}
}
