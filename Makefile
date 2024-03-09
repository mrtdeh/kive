


SHELL := /bin/bash

.PHONY: all test clean

all: test build


test:
	@go clean -testcache && go test -v ./test/unit/*


test-priety:
	@source ./scripts/test/colorize.sh && \
	go clean -testcache && DEBUG=1 ERROR=1 WARN=1 go test -v ./test/unit/* | \
	mygrep -E ok green | \
	mygrep -E PASS green | \
	mygrep -E DEBUG debug | \
	mygrep -E WARN yellow | \
	mygrep -E ERROR red | \
	mygrep -E INFO blue | \
	mygrep -P "(?<=::).*?(?=::)" grey
	
test-integration:
	@source ./scripts/test/colorize.sh && \
	go clean -testcache && go test -v ./test/integration/* | \
	mygrep -E ok green | \
	mygrep -E PASS green | \
	mygrep -E FAIL red 


docker-test:
	./scripts/docker-go-test.sh



build: clean
	go build -v -o bin/centor  ./main.go

build-gc: clean
	go build -gcflags '-m -l' -v -o bin/centor  ./main.go

protoc:
	protoc --go-grpc_out=require_unimplemented_servers=false:./proto/ ./proto/*.proto --go_out=./proto



docker-clean:
	docker image prune -f
	
docker-build: docker-clean
	docker build . --tag mrtdeh/kive
docker-up:
	docker compose -p dc1 -f ./docker-compose-dc1.yml up --force-recreate --build -d
	docker compose -p dc2 -f ./docker-compose-dc2.yml up --force-recreate -d
	docker compose -p dc3 -f ./docker-compose-dc3.yml up --force-recreate -d
	docker compose -p dc4 -f ./docker-compose-dc4.yml up --force-recreate -d

docker-down-all:
	docker compose -p dc1 -f ./docker-compose-dc1.yml down 
	docker compose -p dc2 -f ./docker-compose-dc2.yml down 
	docker compose -p dc3 -f ./docker-compose-dc3.yml down 
	docker compose -p dc4 -f ./docker-compose-dc4.yml down 
	docker compose -p dc1 -f ./docker-compose-with-envoy.yml down 



clean:
	rm -f ./bin/*