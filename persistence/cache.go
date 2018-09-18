package persistence

import (
	"time"
)

//of course that currentRequestCount has an additional field that could be used for the global accumulatedRequestCount
// however, that would be mixing up different concerns: the persistence.RequestCount.accumulatedRequestCount field is meant for
//internal use of the doublylinkedlist only, so that it can calculate the current accumulated values for the relevant time frame
// and therefore not exported.
//thus, a specific counter is added to the request cache.
//TODO names should be used for this differentiation.
type Cache struct {
	RequestCount
	AccumulatedRequestCount int
}

//TODO test
//TODO DOC why the increments?
func NewCache(timestamp time.Time, totalAccumulated int) Cache {
	requestCount := RequestCount{Timestamp: timestamp}
	requestCount.Increment()
	return Cache{
		RequestCount:            requestCount,
		AccumulatedRequestCount: totalAccumulated + 1,
	}
}

//TODO tests
func (c *Cache) Increment() {
	c.RequestCount.Increment()
	c.AccumulatedRequestCount++
}
