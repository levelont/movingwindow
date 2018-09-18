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

func (r RequestCount) Dump() string {
	return fmt.Sprintf("{timestamp:%v, requestsCount:%v, accumulatedRequestCount:%v}", r.Timestamp.String(), r.Count, r.Accumulated)
}

func (r *RequestCount) Increment() {
	r.Count++
	r.Accumulated++
}

/* Determines if the provided timestamp and that of the receiver are considered to be equal by truncating the time
units outside of the provided precision.
*/
func (r RequestCount) CompareTimestampWithPrecision(t time.Time, precision time.Duration) bool {
	return r.Timestamp.Truncate(precision) == t.Truncate(precision)
}

type requestCountNode struct {
	data  RequestCount
	left  *requestCountNode
	right *requestCountNode
}

/*
Checks if the timestamp of the receiver is within the provided duration before the reference. Used to determine if the
receiver node is within the persistence time frame of the reference
*/
func (node requestCountNode) WithinDurationBefore(duration time.Duration, precision time.Duration, reference RequestCount) (bool, time.Duration) {
	difference := reference.Timestamp.Sub(node.data.Timestamp).Truncate(precision)
	return difference.Nanoseconds() <= duration.Nanoseconds(), difference
}

/* Implements a doubly linked list. Used to store the timestamp of requests and their request counts.
Using a doubly linked list for this purpose has the following advantages:
- adding new nodes has constant time complexity O(1): just link a new node to the tail and reset the tail.
- calculating the accumulated request count for the entire persistence time frame has complexity O(n(t)-k), where
  n(t) is the number of nodes on the structure at time t and k is the number of nodes from the structure that no longer
  are within the persistence time frame.

The workflow is as follows:
- New request calculator is instantiated without any nodes.
- New nodes are added to the doubly linked list via AppendToTail()
- The total request count can be calculated via UpdateTotals().
- Update totals will take care of deleting nodes outside of the persistence timeframe by means of FrontDiscardUntil()
- The result of the UpdateTotals() operation will be available via TotalAccumulatedRequestCount

This structure is not safe for concurrent usage. The consumer is responsible for synchronizing all access to it.
*/
type RequestCounter struct {
	head *requestCountNode
	tail *requestCountNode
}

/*Used as a data container of a requestCounter, in particular for serialization purposes.
 */
type requestCountList []RequestCount

// In order to serialize a requestCounter, all data from memory needs to be retrieved, stored in a container and
// serialized as such. Only the data is needed. Current memory locations are not necessary,
func (r RequestCounter) getNodes() requestCountList {
	//TODO reasonable capacity? cache it at the doublylinkedList?
	nodes := make(requestCountList, 0, 100)
	currentNode := r.head
	for currentNode != nil {
		nodes = append(nodes, currentNode.data)
		currentNode = currentNode.right
	}

	return nodes
}

/* Test instrumentation
 */
func (r RequestCounter) dump() string {
	var b strings.Builder
	for _, node := range r.getNodes() {
		b.WriteString(node.Dump())
	}
	return b.String()
}

/* Upon deserialize data from disk, what results is a list containing the data that would make up a requestCounter.
This function takes care of transforming the data back into a doubly linked list
*/
func (values requestCountList) ToRequestCounter() RequestCounter {
	var list RequestCounter
	for _, value := range values {
		list = list.AppendToTail(value)
	}

	return list
}

/* Creates an new node with the provided data. Sets it both as the right node of the current tail and as the new tail
of the list. If the list is empty, the new node will also be promoted to be the head of the list.
*/
func (list RequestCounter) AppendToTail(data RequestCount) RequestCounter {
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
*/
func (list RequestCounter) frontDiscardUntil(lastNodeToDiscard *requestCountNode) RequestCounter {
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
func (list RequestCounter) UpdateTotals(reference RequestCount, timeFrame time.Duration) RequestCounter {
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
			list = list.frontDiscardUntil(currentNode)
			log.Printf("UPDATETOTALS: List is now '%+v'\n", list.dump())
			break

		}
	}

	return list
}

/*
Assumes that UpdateTotals was called before and that the head node contains the totals.
*/
func (list RequestCounter) TotalAccumulatedRequestCount() int {
	if list.head != nil {
		return list.head.data.Accumulated
	} else {
		return 0
	}
}
