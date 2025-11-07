build:
	go build -o bin/psql-transporter ./cmd/psql-transporter

run:
	go run ./cmd/psql-transporter

fmt:
	go fmt ./...

lint:
	go vet ./...