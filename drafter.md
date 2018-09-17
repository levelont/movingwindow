# Problem description

Using only the standard library, create a Go HTTP server that on each request responds with a counter of the total
number of requests that it has received during the previous 60 seconds (moving window).
The server should continue to the return the correct numbers after restarting it, by persisting data to a file.

# Design:
- Initialize
	[TODO] read state from file
	[TODO] how to initialiye cache and persistence struct for the very first round?
- Incoming requests have a timestamp.
	- Truncate to seconds - that's the minimum unit we will work with.
- Compare incoming timestamp with the held cache
	- Cache contains last updated timestamp and counter of requests that have come in that lapse second
- If cache is empty
	- set timestamp to that of the request and count to 1
	- - dispatch response
		getDataForLast59() + Cache.totalRequests
		getDataForLast59() = 0 if HEAD == nil
- If the timestamp is equal
 	[DEPRECATED] - First, read the value and dispatch the response right away.
 		[DEPRECATED] - it is important that the read is done first because a read has no contention - no waiting time.
 		[DEPRECATED] - but it is also an issue: reads are much faster than writes, so some responses might be dispatched
 		      with wrong values
 	- increase cache counter by one
		- This should be some sort of locked operation: many concurrent requests may come at the same time.
		- the increase  operation should return the value after the increase
	- get the totals from the persistence structure
		- the total 60 second time span information will be made up by the cache and the persistence structure
			- the structure should be a doubled linked list for:
				- easy attachment of new request counts on the tail (newest/latest timestamps)
				- easy cleanup of discarded nodes that are no longer part of the 60 second time span
			- the structure will be composed of the same struct types that the cache has
				- this allows for quick updates of the structure
			- each node should keep a totalRequestsCountSoFar value for quick retrieval of the totals
				- moving from the tail back to past requests,it should keep an accumulated of requests
				  across time
				- it should be updated with every node insertion
		- it should provide with a method addNewRequestCount which:
			- links the cache to the existing requestCounts structure
				- if structure is empty
					HEAD = cache
					// default: cache.previous = nil
					cache.totalRequestsSoFar = cache.totalRequests
					// default: cache.next = nil
					TAIL = cache
				- else
					TAIL.next = cache
					cache.previous = TAIL
					cache.totalRequestsSoFar = cache.totalRequests
					// default: cache.next = nil
					TAIL = cache
			- traverses the structure by checking if the previous node is in a 60 second interval with the
			  last request timestamp as the reference
			  * start from tail.previous
			  * continue until end of the doubly linked list is reached
			  - while current != nil
				  - if newRequest.timestamp - currentNode.timestamp <= 60s
					currentNode.totalRequestsCountSoFar = currentNode.totalRequests + nextNode.totalRequestsCountSoFar
					current = current.previous
				  - else
					if it isn't, the next node is set as the head of the doubly linked list, links are broken and the remaining
					nodes are discarded
					- nextNode is the last node that was inside the 60s interval
					- everything else (HEAD -> currentNode) can be discarded
					current.next.previous = nil // break link from last valid node in the interval to the current node
					TOBEDISCARDED = HEAD
					HEAD = current.next
					current.next = nil
					discardLeftOvers(TOBEDISCARDED) // is this even necessary?
									// can be async
					TOBEDISCARDED = nil
					current = nil
		- it should provide with a method getTotalRequestCountSoFar
			- because of the mechanics of addNewRequestCount, the total will be the value of
			  totalRequestsCountSoFar stored in the head of the doubly linked list.
	- dispatch response
		getDataForLast59() + Cache.totalRequests
		TODO good naming convention to differentiate between totalRequests in the current split second and
		TODO totalRequestsSoFar in the entire timeframe from cache til current node in the doubly linked list.
- If the timestamp differs
	- it will differ by at least one second
	- add current cache to the list by means of addNewRequestCount
		TODO better name for the method
	- reset cache to timestamp of new request and totalRequests = 1
		- reset the cache pointer to a new one with those values!
	- get the total request count from the persistence structure
		getTotalRequestCountSoFar()
	- dispatch response
		getTotalRequestCountSoFar() + Cache.totalRequests

discardLeftOvers(TOBEDISCARDED)
	current = TOBEDISCARDED
	while current != nil
		- nextNodeInDoublyLinkedList = current.next
		- current.previous = nil
		- current.next = nil
		- current =  nextNodeInDoublyLinkedList

detectInterrupt()
	-persist the doublylinkedlist from HEAD to TAIL into persistence file
	-persist cache appending it to the persistence file

# ACTION PLAN

- [X] define node struct
- [X] define data struct
- [X] define node with data struct // composition
- [X] define doublylinked list
	- struct with head and tail 'node with data' nodes
	- [X] add node operation
	- [X] update totals operation
	- [X] discard nodes operation
		- better to define it with an open interval
		- requires HEAD and NEWHEAD as parameters
		- nodes, starting from HEAD will be removed one by one until NEWHEAD is reached
		- NEWHEAD will be the new HEAD of the doubly linked list after the operations
- [X] extract discard node subset logic from update totals
    - [X] replace discard logic @ UpdateTotals with new function
- [X] get total accumulated requests
- [X] http server layer
    - [INVALID] retrieve timestamp from request
- [ ] signal manager MUST close the communication channels!
- [ ] logging with different components and visibilities
- [X] response writer
- [ ] high availability 
    - [ ] identify serialization points
    - [ ] write state to file
    - [ ] read state from file
- [INVALID] concurrency concerns
    - [INVALID] locked write access to cache?
    - [INVALID] locked write access to doublyLinkedList?
- [ ] refactorings
    - [ ] structure of http_test
    - [ ] structure of main
    - [ ] wrapper for doublylinkedList
        - [ ] a better name!
- [ ] TODOs 
- [ ] precision of the algorithm - allow for smaller than 1s
- [ ] linter warnings

