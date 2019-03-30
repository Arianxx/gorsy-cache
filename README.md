```
                                                      __       
   ____ _____  ____________  __      _________ ______/ /_  ___ 
  / __ `/ __ \/ ___/ ___/ / / /_____/ ___/ __ `/ ___/ __ \/ _ \
 / /_/ / /_/ / /  (__  ) /_/ /_____/ /__/ /_/ / /__/ / / /  __/
 \__, /\____/_/  /____/\__, /      \___/\__,_/\___/_/ /_/\___/ 
/____/                /____/                                   
```

Gorsy is a concurrency-safe in-memory k/v cache store implemented by Golang that supports the lru, lfu, arc algorithm etc.

### Example
```golang
package main

import (
	"fmt"
	"github.com/arianxx/gorsy-cache"
)

func loader(_ interface{}) (interface{}, error) {
	return "baka!", nil
}

func beforeEvict(key, value interface{}) {
	fmt.Printf("%v: %v was be removed\n", key, value)
}

func main() {
	builder, err := gorsy_cache.NewBuilder(gorsy_cache.LFU, 10)
	if err != nil {
		panic("build cache error: " + err.Error())
	}
	cache := builder.
		SetName("rocket").
		SetDefaultExpiration(gorsy_cache.DefaultExpiration).
		SetLoaderFunc(loader).
		SetBeforeEvictedFunc(beforeEvict).
		Build()
	cache.Set(1, 2)
	cache.Set(2, 3)
	fmt.Println(cache.Get(1))
	fmt.Println(cache.Get(2))
	fmt.Println(cache.GetOnlyPresent(3))
	cache.Remove(1)
	fmt.Println(cache.Get(1))
}

```