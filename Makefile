build:
	CGO_ENABLED=0 go build -o ./bin/neveralone ./cmd/neveralone/.

run:
	go run ./cmd/neveralone/.

dockerize:
	docker build -f ./cmd/neveralone/Dockerfile -t neveralone:dev .