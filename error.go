package gorsy_cache

import "fmt"

type KeyNotFoundError struct {
	Name string
	Key  interface{}
	Err  error
}

func (e *KeyNotFoundError) Error() string {
	s := "`%s`: key `%s` not found in the cache store"
	if e.Err != nil {
		s += " with loader function error: " + e.Err.Error()
	}
	return fmt.Sprintf(s, e.Name, e.Key)
}
