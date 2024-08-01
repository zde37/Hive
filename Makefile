run:
	go run cmd/main.go

test:
	go test -v -cover -count 1 ./...

mock-client:
	mockgen -package mocked -destination internal/mocks/client.go github.com/zde37/Hive/internal/ipfs Client

mock-handler:
	mockgen -package mocked -destination internal/mocks/handler.go github.com/zde37/Hive/internal/handler Handler

.PHONY: run test mock-client mock-handler