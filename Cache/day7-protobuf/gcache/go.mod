module gcache

go 1.21.3

require (
	google.golang.org/protobuf v1.33.0
	lru v0.0.0
)

replace lru => ./lru
