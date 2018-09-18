package persistence

import (
	"fmt"
	"log"
	"strings"
	"time"
)

/*Number of requests associated to the specific point in time indicated by 'Timestamp'.
'Count' accumulates the number of received requests that came at the same time, according to the precision of the
algorithm. When calculating totals from a given reference, 'Accumulated' sums the number of requests received from the
timestamp of the reference until 'Timestamp'
Fields need be exported for encoding purposes.
*/
type RequestCount struct {
	Timestamp   time.Time
	Count       int
	Accumulated int
}

func (r RequestCount) Empty() bool {
	return r.Timestamp.IsZero()
}

//TODO test function?
func (r RequestCount) Dump() string {
	return fmt.Sprintf("{timestamp:%v, requestsCount:%v, accumulatedRequestCount:%v}", r.Timestamp.String(), r.Count, r.Accumulated)
}

//TODO test coverage
//TODO analyze test coverage of entire project
func (r *RequestCount) Increment() {
	r.Count++
	r.Accumulated++
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

type requestCountList []RequestCount

// in order to serialize this, i'd have to traverse and serialize each of the nodes
// only the data is needed. all memory locations are not necessary, so I can work solely with requestCounts
func (r RequestCountDoublyLinkedList) getNodes() requestCountList {
	//TODO reasonable capacity? cache it at the doublylinkedList?
	nodes := make(requestCountList, 0, 100)
	currentNode := r.head
	for currentNode != nil {
		nodes = append(nodes, currentNode.data)
		currentNode = currentNode.right
	}

	return nodes
}

func (r RequestCountDoublyLinkedList) dump() string {
	var b strings.Builder
	for _, node := range r.getNodes() {
		b.WriteString(node.Dump())
	}
	return b.String()
}

func (values requestCountList) BuildDoublyLinkedList() RequestCountDoublyLinkedList {
	var list RequestCountDoublyLinkedList
	for _, value := range values {
		list = list.AppendToTail(value)
	}

	return list
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
		list.head.left = nil
	}

	for currentNode != nil {
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
Backward-traverses the list starting from the tail. Checks that each node is within the provided timeframe in seconds
before the reference.
Nodes with timestamps in the time frame will get their accumulatedRequestCount value updated to the sum of their
requestCount and the accumulated of the node right to them.
As such, the total of accumulated requests received between the reference and the previous 60 seconds will be the
accumulatedRequestCount value of the head of the list.
Nodes outside of the time frame will be discarded from the list.
*/
func (list RequestCountDoublyLinkedList) UpdateTotals(reference RequestCount, timeFrame time.Duration) RequestCountDoublyLinkedList {
	currentNode := list.tail
	log.Printf("UPDATETOTALS: Updating totals with reference '%v' and accepted timeframe '%v'\n", reference, timeFrame)
	log.Printf("UPDATETOTALS: List state '%v'\n", list.dump())
	for currentNode != nil {
		log.Printf("UPDATETOTALS: At current node with data '%v'\n", currentNode.data)
		if withinTimeFrame, _ := currentNode.WithinDurationBefore(timeFrame, time.Second, reference); withinTimeFrame {
			if currentNode.right != nil {
				currentNode.data.Accumulated = currentNode.data.Count + currentNode.right.data.Accumulated
			} else {
				currentNode.data.Accumulated = currentNode.data.Count
			}
			log.Printf("UPDATETOTALS: Current node is within the accepted timeframe. It's data was updated to: '%v'\n", currentNode.data)
			currentNode = currentNode.left
		} else {
			log.Println("UPDATETOTALS: Current node is NOT within the accepted timeframe. Nodes wil be discarded from list.")
			log.Printf("UPDATETOTALS: List state '%v'\n", list.dump())
			list = list.FrontDiscardUntil(currentNode)
			log.Printf("UPDATETOTALS: List is now '%+v'\n", list.dump())
			break

		}
	}

	return list
}

/*
Assumes that UpdateTotals was called right before.
*/
func (list RequestCountDoublyLinkedList) TotalAccumulatedRequestCount() int {
	if list.head != nil {
		return list.head.data.Accumulated
	} else {
		return 0
	}
}

func (r RequestCount) CompareTimestampWithPrecision(t time.Time, precision time.Duration) bool {
	return r.Timestamp.Truncate(precision) == t.Truncate(precision)
}
