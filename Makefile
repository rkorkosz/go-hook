.EXPORT_ALL_VARIABLES:
	CGO_ENABLED=0

http-single:
	go build -o hts ./cmd/http/single/main.go

http-multi:
	go build -o htm ./cmd/http/multi/main.go

test:
	go test -v ./...

docker:
	docker-compose up -d --build --scale go-hook=3

lint:
	golangci-lint run

build: http-single http-multi
