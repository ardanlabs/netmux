DT := $(shell date +%Y.%m.%d.%H%M%S)

version:
	echo $(DT) > ./app/server/version
	echo $(DT) > ./app/nx/version
	cp semver ./app/server/semver
	cp semver ./app/nx/semver

proto:
	protoc --go_out=.  --go-grpc_out=.  ./misc/proto/agent.proto
	protoc --go_out=.  --go-grpc_out=.  ./misc/proto/proxy.proto

bin: version
	go build -o ./bin/nx ./cmd/nx

docker-bin:
	echo $(DT) > ./app/server/version
	GOOS=linux go build -o ./bin/server-aarch64 ./cmd/server
	GOOS=linux GOARCH=amd64 go build -o ./bin/server-x86_64 ./cmd/server

docker-amd64: version
	echo $(DT) > ./app/server/version
	GOOS=linux GOARCH=amd64 go build -o ./bin/server ./cmd/server
	docker build --platform linux/amd64 -t digitalcircle/netmux:amd64 -f ./misc/docker/Dockerfile .

docker-arm64: version
	GOOS=linux go build -o ./bin/server ./cmd/server
	docker build --platform linux/arm64 -t digitalcircle/netmux:arm64 -f ./misc/docker/Dockerfile .

docker-push:
	docker push digitalcircle/netmux:arm64
	docker push digitalcircle/netmux:amd64

docker-local: version docker-amd64 docker-arm64

docker-all: docker-local docker-push

local: docker-arm64 bin install-agent

install-agent:
	- sudo rm /usr/local/bin/nx
	sudo go build -o /usr/local/bin/nx ./cmd/nx

uninstall-agent:
	sudo rm /usr/local/bin/nx
