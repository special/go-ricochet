all:
	go install github.com/go-ricochet

test:
	go test -v github.com/s-rah/go-ricochet/...

cover:
	go test github.com/s-rah/go-ricochet/... -cover

