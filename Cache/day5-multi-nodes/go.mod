module example

go 1.21.3

require gcache v0.0.0

replace gcache => ./gcache

require lru v0.0.0 // indirect

replace lru => ./gcache/lru
