package persistence

import (
	"fmt"
	"strconv"
	"strings"
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
		newNode.left = list.tail
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

func (list requestCountDoublyLinkedList) DumpBackwards() string {
	var result strings.Builder
	currentNode := list.tail
	for {
		if currentNode == nil {
			break
		}
		result.WriteString(strconv.Itoa(currentNode.data.requestsCount))
		currentNode = currentNode.left
	}

	return result.String()
}

func (list requestCountDoublyLinkedList) UpdateTotals(reference requestCount) requestCountDoublyLinkedList {
	currentNode := list.tail.left
	for {
		if currentNode == nil {
			break
		}

		if within60SecondsFromReference, _ := currentNode.WithinDurationFrom(time.Duration(60)*time.Second, time.Second, reference); within60SecondsFromReference {
			currentNode.data.accumulatedRequestCount = currentNode.data.requestsCount + currentNode.right.data.accumulatedRequestCount
			currentNode = currentNode.left
		} else {
			nodeToBeDiscarded := list.head
			list.head = currentNode.right
			currentNode.right = nil
			currentNode.left = nil
			currentNode = nil

			//discard
			for {
				if nodeToBeDiscarded == nil {
					break
				}

				temp := nodeToBeDiscarded.right
				nodeToBeDiscarded.left = nil
				nodeToBeDiscarded.right = nil
				nodeToBeDiscarded = temp
			}
		}
	}

	return list
}
