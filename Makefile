.EXPORT_ALL_VARIABLES:
	CGO_ENABLED=0

http-single:
	go build -o hts ./cmd/http/single/main.go

http-multi:
	go build -o htm ./cmd/http/multi/main.go

test:
	go test -v ./...

docker:
	docker-compose up -d --scale go-hook=3

.PHONY: http-single http-multi