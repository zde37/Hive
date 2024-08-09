IPFS_CONTAINER_NAME=ipfs_host
IPFS_STAGING=$(HOME)/.ipfs/staging
IPFS_DATA=$(HOME)/.ipfs/data
IPFS_PROFILE=server

run:
	go run cmd/main.go

test:
	go test -v -cover -timeout 600s -count 1 ./...

mock-handler:
	mockgen -package mocked -destination internal/mocks/handler.go github.com/zde37/Hive/internal/handler Handler
	
ipfs-init:
	docker run -d --name $(IPFS_CONTAINER_NAME) \
	-e IPFS_PROFILE=$(IPFS_PROFILE) \
	-v $(IPFS_STAGING):/export \
	-v $(IPFS_DATA):/data/ipfs \
	-p 4001:4001 -p 4001:4001/udp \
	-p 127.0.0.1:8080:8080 \
	-p 127.0.0.1:5001:5001 \
	ipfs/kubo:latest

ipfs-run:
	@if [ -n "$(cmd)" ]; then \
		docker exec $(IPFS_CONTAINER_NAME) $(cmd); \
	else \
		echo "Error: Missing 'cmd' variable\n" >&2; \
		echo "Usage: make ipfs-run cmd=\"ipfs swarm peers\"\n" >&2; \
		exit 1; \
	fi	

ipfs-logs:
	docker logs -f $(IPFS_CONTAINER_NAME)

ipfs-start:
	docker start $(IPFS_CONTAINER_NAME)

ipfs-stop:
	docker stop $(IPFS_CONTAINER_NAME)

ipfs-rm:
	docker rm $(IPFS_CONTAINER_NAME)
	rm -rf $(IPFS_DATA)
	rm -rf $(IPFS_STAGING)

.PHONY: run test mock-handler ipfs_rm ipfs_stop ipfs-start ipfs-logs ipfs-run ipfs-init