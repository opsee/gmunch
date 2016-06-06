APPENV ?= testenv
PROJECT := $(shell basename $$PWD)
REV ?= latest

all: build

clean:
	rm -fr target bin pkg

fmt:
	@gofmt -w ./

deps:
	docker-compose up -d
	
proto:
	protoc --proto_path=./vendor:. --go_out=plugins=grpc:. *.proto

build: deps $(APPENV)
	govendor build -v ./...

run_worker: deps $(APPENV)
	docker run \
	  --link gmunch_etcd:etcd \
		--env-file ./$(APPENV) \
		-e AWS_DEFAULT_REGION \
		-e AWS_ACCESS_KEY_ID \
		-e AWS_SECRET_ACCESS_KEY \
		-p 9105:9105 \
		--rm \
		quay.io/opsee/$(PROJECT):$(REV)

.PHONY: docker run_worker run_server migrate clean all