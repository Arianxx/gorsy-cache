package gorsy_cache

import (
	"fmt"
	"time"
)

var purging = make(map[*Cache]chan struct{})

func StartPurge(c *Cache, d time.Duration) error {
	if _, ok := purging[c]; ok {
		return fmt.Errorf("%v has been started to purge", c)
	}

	purging[c] = make(chan struct{})
	go func() {
		t := time.NewTicker(d * time.Second)
		for {
			select {
			case <-purging[c]:
				t.Stop()
				delete(purging, c)
				return
			case <-t.C:
				(*c).CleanExpired()
			}
		}
	}()

	return nil
}

func StopPurge(c *Cache) {
	if _, ok := purging[c]; ok {
		purging[c] <- struct{}{}
	}
}
