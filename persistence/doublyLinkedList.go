package persistence

import (
	"fmt"
	"time"
)

type requestCount struct {
	timestamp               time.Time
	requestsCount           int
	accumulatedRequestCount int
}

func (r requestCount) Dump() string {
	return fmt.Sprintf("{timestamp:%v, requestsCount:%v, accumulatedRequestCount:%v}\n", r.timestamp.String(), r.requestsCount, r.accumulatedRequestCount)
}

type requestCountNode struct {
	data  requestCount
	left  *requestCountNode
	right *requestCountNode
}

/*
Checks if the timestamp of the receiver is within the provided duration before the reference.
*/
func (node requestCountNode) WithinDurationBefore(duration time.Duration, precision time.Duration, reference requestCount) (bool, time.Duration) {
	difference := reference.timestamp.Sub(node.data.timestamp).Truncate(precision)
	return difference.Nanoseconds() <= duration.Nanoseconds(), difference
}

type requestCountDoublyLinkedList struct {
	head *requestCountNode
	tail *requestCountNode
}

/*
Creates an new node with the provided data and sets it both as the right node of the current tail and as the new tail of the list
*/
func (list requestCountDoublyLinkedList) AppendToTail(data requestCount) requestCountDoublyLinkedList {
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
func (list requestCountDoublyLinkedList) FrontDiscardUntil(lastNodeToDiscard *requestCountNode) requestCountDoublyLinkedList {
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
Backward-traverses the list starting from the node left to the tail. Checks that each node is within 60 seconds before
the reference.
Nodes with timestamps in the time frame will get their accumulatedRequestCount value updated to the sum of their
requestCount and the accumulated of the node right to them.
As such, the total of accumulated requests received between the reference and the previous 60 seconds will be the
accumulatedRequestCount value of the head of the list.
Nodes outside of the time frame will be discarded from the list.
*/
func (list requestCountDoublyLinkedList) UpdateTotals(reference requestCount) requestCountDoublyLinkedList {
	currentNode := list.tail.left
	for {
		if currentNode == nil {
			break
		}

		if within60SecondsFromReference, _ := currentNode.WithinDurationBefore(time.Duration(60)*time.Second, time.Second, reference); within60SecondsFromReference {
			currentNode.data.accumulatedRequestCount = currentNode.data.requestsCount + currentNode.right.data.accumulatedRequestCount
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
func (list requestCountDoublyLinkedList) TotalAccumulatedRequestCount() int {
	return list.head.data.accumulatedRequestCount
}
