package gorsy_cache

import "time"

type baseItem struct {
	key, value interface{}
	expiration *time.Time
}

func (s *baseItem) isExpired() bool {
	if s.expiration == nil {
		return false
	}

	return s.expiration.Before(time.Now())
}

func (s *baseItem) setExpiration(expiration time.Duration, c *baseCache) {
	if expiration == DefaultExpiration {
		expiration = c.Expiration
	}
	if expiration != NoExpiration {
		t := time.Now().Add(expiration * time.Second)
		s.expiration = &t
	} else {
		s.expiration = nil
	}
}
