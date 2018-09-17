package persistence

import (
	"fmt"
	"time"
)

//todo names: see messed up usage @startCommunicationProcessor
// maybe just count and accumulated?
//TODO hide accumulatedRequestCount from json encoding
type RequestCount struct {
	Timestamp               time.Time
	RequestsCount           int
	accumulatedRequestCount int
}

func (r RequestCount) Empty() bool {
	return r.Timestamp.IsZero()
}

func (r RequestCount) Dump() string {
	return fmt.Sprintf("{timestamp:%v, requestsCount:%v, accumulatedRequestCount:%v}\n", r.Timestamp.String(), r.RequestsCount, r.accumulatedRequestCount)
}

//TODO test coverage
//TODO analyze test coverage of entire project
func (r *RequestCount) Increment() {
	r.RequestsCount++
	r.accumulatedRequestCount++
}

type requestCountNode struct {
	data  RequestCount
	left  *requestCountNode
	right *requestCountNode
}

/*
Checks if the timestamp of the receiver is within the provided duration before the reference.
*/
//TODO no real need to have the reference as a requestCount. Can be a simple timestamp
func (node requestCountNode) WithinDurationBefore(duration time.Duration, precision time.Duration, reference RequestCount) (bool, time.Duration) {
	difference := reference.Timestamp.Sub(node.data.Timestamp).Truncate(precision)
	return difference.Nanoseconds() <= duration.Nanoseconds(), difference
}

//TODO name
type RequestCountDoublyLinkedList struct {
	head *requestCountNode
	tail *requestCountNode
}

/*
Creates an new node with the provided data and sets it both as the right node of the current tail and as the new tail of the list
*/
func (list RequestCountDoublyLinkedList) AppendToTail(data RequestCount) RequestCountDoublyLinkedList {
	//new node with provided data
	newNode := requestCountNode{data: data}
	if list.head == nil {
		list.head = &newNode
		list.tail = &newNode
	} else {
		// new node as next of tail
		list.tail.right = &newNode
		newNode.left = list.tail
		// tail = next of tail
		list.tail = &newNode
	}

	return list
}

/*
Discard all nodes between head and lastNodeToDiscard from the list. Assumes that lastNodeToDiscard is part of the list
Specifying tail as lastNodeToDiscard discards all nodes from the list. The resulting list will have head = tail = nil.
Otherwise, lastNodeToDiscard.right will be the new head of the list
//TODO no need for this to be exported
//TODO after creating a workflow wrapper, not much will be needed to be exported
*/
func (list RequestCountDoublyLinkedList) FrontDiscardUntil(lastNodeToDiscard *requestCountNode) RequestCountDoublyLinkedList {
	currentNode := list.head
	if lastNodeToDiscard == list.tail {
		list.head = nil
		list.tail = nil
	} else {
		list.head = lastNodeToDiscard.right
	}
	for {
		if currentNode == nil {
			break
		}

		atLastNode := false
		if currentNode == lastNodeToDiscard {
			atLastNode = true
		}

		temp := currentNode.right
		currentNode.left = nil
		currentNode.right = nil
		currentNode = temp

		if atLastNode {
			break
		}
	}

	return list
}

/*
Backward-traverses the list starting from the node left to the tail. Checks that each node is within the provided
timeframe in seconds before the reference.
Nodes with timestamps in the time frame will get their accumulatedRequestCount value updated to the sum of their
requestCount and the accumulated of the node right to them.
As such, the total of accumulated requests received between the reference and the previous 60 seconds will be the
accumulatedRequestCount value of the head of the list.
Nodes outside of the time frame will be discarded from the list.
*/
func (list RequestCountDoublyLinkedList) UpdateTotals(reference RequestCount, timeFrame time.Duration) RequestCountDoublyLinkedList {
	currentNode := list.tail.left
	for {
		if currentNode == nil {
			break
		}

		if withinTimeFrame, _ := currentNode.WithinDurationBefore(timeFrame, time.Second, reference); withinTimeFrame {
			currentNode.data.accumulatedRequestCount = currentNode.data.RequestsCount + currentNode.right.data.accumulatedRequestCount
			currentNode = currentNode.left
		} else {
			list = list.FrontDiscardUntil(currentNode)
			break

		}
	}

	return list
}

/*
Assumes that UpdateTotals was called right before.
*/
func (list RequestCountDoublyLinkedList) TotalAccumulatedRequestCount() int {
	return list.head.data.accumulatedRequestCount
}

func (r RequestCount) CompareTimestampWithPrecision(t time.Time, precision time.Duration) bool {
	return r.Timestamp.Truncate(precision) == t.Truncate(precision)
}
