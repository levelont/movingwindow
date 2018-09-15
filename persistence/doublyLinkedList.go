package persistence

import (
	"strconv"
	"strings"
	"time"
)

type requestCount struct {
	timestamp               time.Time
	requestsCount           int
	accumulatedRequestCount int
}

type requestCountNode struct {
	data  requestCount
	left  *requestCountNode
	right *requestCountNode
}

func (node requestCountNode) WithinDurationFrom(duration time.Duration, precision time.Duration, reference requestCount) (bool, time.Duration) {
	difference := reference.timestamp.Sub(node.data.timestamp).Truncate(precision)
	return difference.Nanoseconds() <= duration.Nanoseconds(), difference
}

type requestCountDoublyLinkedList struct {
	head *requestCountNode
	tail *requestCountNode
}

func (list requestCountDoublyLinkedList) AppendToTail(data requestCount) requestCountDoublyLinkedList {
	//new node with provided data
	newNode := requestCountNode{data: data}
	if list.head == nil {
		list.head = &newNode
		list.tail = &newNode
	} else {
		// new node as next of tail
		list.tail.right = &newNode
		// tail = next of tail
		list.tail = &newNode
	}

	return list
}

func (list requestCountDoublyLinkedList) Dump() string {
	var result strings.Builder
	currentNode := list.head
	for {
		if currentNode == nil {
			break
		}
		result.WriteString(strconv.Itoa(currentNode.data.requestsCount))
		currentNode = currentNode.right
	}

	return result.String()
}

func (list requestCountDoublyLinkedList) UpdateTotals(reference requestCount) {
	currentNode := list.tail.left
	for {
		if currentNode == nil {
			break
		}

		//TODO LEFT HERE
	}
}
