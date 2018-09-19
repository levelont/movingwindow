package persistence

import (
	"fmt"
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	fmt.Println(time.Date(2006, 01, 02, 19, 00, 59, 0, time.UTC).Sub(time.Date(2006, 01, 02, 18, 59, 58, 999000001, time.UTC)).String())
}
