package main

import (
	"fmt"
	"github.com/arianxx/gorsy-cache"
)

func loader(key interface{}) (interface{}, error) {
	return "baka!", nil
}

func beforeEvict(key, value interface{}) {
	fmt.Printf("%v: %v was be removed\n", key, value)
}

func main() {
	builder, err := gorsy_cache.NewBuilder(gorsy_cache.SIMPLE, 10)
	if err != nil {
		panic("build cache: " + err.Error())
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
	fmt.Println(cache.Get(3))
	fmt.Println(cache.GetOnlyPresent(3))
	cache.Remove(2)
	fmt.Println(cache.Get(2))
}
