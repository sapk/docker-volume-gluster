#Inspired from : https://github.com/littlemanco/boilr-makefile/blob/master/template/Makefile, https://github.com/geetarista/go-boilerplate/blob/master/Makefile, https://github.com/nascii/go-boilerplate/blob/master/GNUmakefile https://github.com/cloudflare/hellogopher/blob/master/Makefile
#PATH=$(PATH:):$(GOPATH)/bin
APP_NAME=docker-volume-gluster
APP_VERSION=$(shell git describe --tags --abbrev=0)
APP_USERREPO=github.com/sapk
APP_PACKAGE=$(APP_USERREPO)/$(APP_NAME)


PLUGIN_USER ?= sapk
PLUGIN_NAME ?= plugin-gluster
PLUGIN_TAG ?= latest
PLUGIN_IMAGE ?= $(PLUGIN_USER)/$(PLUGIN_NAME):$(PLUGIN_TAG)

GIT_HASH=$(shell git rev-parse --short HEAD)
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
DATE := $(shell date -u '+%Y-%m-%d-%H%M-UTC')
PWD=$(shell pwd)

ARCHIVE=$(APP_NAME)-$(APP_VERSION)-$(GIT_HASH).tar.gz
#DEPS = $(go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
LDFLAGS = \
  -s -w \
  -X main.Version=$(APP_VERSION) -X main.Branch=$(GIT_BRANCH) -X main.Commit=$(GIT_HASH) -X main.BuildTime=$(DATE)

FAKE_GOPATH = $(PWD)/.gopath
FAKE_PACKAGE = $(FAKE_GOPATH)/src/$(APP_PACKAGE)

GO15VENDOREXPERIMENT=1
DOC_PORT = 6060
#GOOS=linux

ERROR_COLOR=\033[31;01m
NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
WARN_COLOR=\033[33;01m

GOPATH ?= $(HOME)/go

all: build compress done

build: deps clean format compile

docker-plugin: docker-rootfs docker-plugin-create

docker-image:
	@echo -e "$(OK_COLOR)==> Docker build image : ${PLUGIN_IMAGE} $(NO_COLOR)"
	docker build -t ${PLUGIN_IMAGE} -f support/docker/Dockerfile .

docker-rootfs: docker-image
	@echo -e "$(OK_COLOR)==> create rootfs directory in ./plugin/rootfs$(NO_COLOR)"
	@mkdir -p ./plugin/rootfs
	@cntr=${PLUGIN_USER}-${PLUGIN_NAME}-${PLUGIN_TAG}-$$(date +'%Y%m%d-%H%M%S'); \
	docker create --name $$cntr ${PLUGIN_IMAGE}; \
	docker export $$cntr | tar -x -C ./plugin/rootfs; \
	docker rm -vf $$cntr
	@echo -e "### copy config.json to ./plugin/$(NO_COLOR)"
	@cp config.json ./plugin/

docker-plugin-create:
	@echo -e "$(OK_COLOR)==> Remove existing plugin : ${PLUGIN_IMAGE} if exists$(NO_COLOR)"
	@docker plugin rm -f ${PLUGIN_IMAGE} || true
	@echo -e "$(OK_COLOR)==> Create new plugin : ${PLUGIN_IMAGE} from ./plugin$(NO_COLOR)"
	docker plugin create ${PLUGIN_IMAGE} ./plugin

docker-plugin-push:
	@echo -e "$(OK_COLOR)==> push plugin : ${PLUGIN_IMAGE}$(NO_COLOR)"
	docker plugin push ${PLUGIN_IMAGE}

docker-plugin-enable:
	@echo -e "$(OK_COLOR)==> Enable plugin ${PLUGIN_IMAGE}$(NO_COLOR)"
	docker plugin enable ${PLUGIN_IMAGE}

set-build:
	@if [ ! -d $(PWD)/.gopath/src/$(APP_USERREPO) ]; then mkdir -p $(PWD)/.gopath/src/$(APP_USERREPO); fi
	@if [ ! -d $(PWD)/.gopath/src/$(APP_PACKAGE) ]; then ln -s $(PWD) $(PWD)/.gopath/src/$(APP_PACKAGE); fi

compile: set-build
	@echo -e "$(OK_COLOR)==> Building...$(NO_COLOR)"
	cd $(FAKE_PACKAGE) && GOPATH=$(FAKE_GOPATH) go build -v -ldflags "$(LDFLAGS)"

release: clean set-build deps format
	@mkdir build
	@echo -e "$(OK_COLOR)==> Building for linux 32 ...$(NO_COLOR)"
	cd $(FAKE_PACKAGE) && GOPATH=$(FAKE_GOPATH) CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -o build/${APP_NAME}-linux-386 -ldflags "$(LDFLAGS)"
	@echo -e "$(OK_COLOR)==> Trying to compress binary ...$(NO_COLOR)"
	@upx --brute  build/${APP_NAME}-linux-386 || upx-ucl --brute  build/${APP_NAME}-linux-386 || echo -e "$(WARN_COLOR)==> No tools found to compress binary.$(NO_COLOR)"

	@echo -e "$(OK_COLOR)==> Building for linux 64 ...$(NO_COLOR)"
	cd $(FAKE_PACKAGE) && GOPATH=$(FAKE_GOPATH) GO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/${APP_NAME}-linux-amd64 -ldflags "$(LDFLAGS)"
	@echo -e "$(OK_COLOR)==> Trying to compress binary ...$(NO_COLOR)"
	@upx --brute  build/${APP_NAME}-linux-amd64 || upx-ucl --brute  build/${APP_NAME}-linux-amd64 || echo -e "$(WARN_COLOR)==> No tools found to compress binary.$(NO_COLOR)"

	@echo -e "$(OK_COLOR)==> Building for linux arm ...$(NO_COLOR)"
	cd $(FAKE_PACKAGE) && GOPATH=$(FAKE_GOPATH) CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -o build/${APP_NAME}-linux-armv6 -ldflags "$(LDFLAGS)"
	@echo -e "$(OK_COLOR)==> Trying to compress binary ...$(NO_COLOR)"
	@upx --brute  build/${APP_NAME}-linux-armv6 || upx-ucl --brute  build/${APP_NAME}-linux-armv6 || echo -e "$(WARN_COLOR)==> No tools found to compress binary.$(NO_COLOR)"

	@echo -e "$(OK_COLOR)==> Building for darwin32 ...$(NO_COLOR)"
	cd $(FAKE_PACKAGE) && GOPATH=$(FAKE_GOPATH) CGO_ENABLED=0 GOOS=darwin GOARCH=386 go build -o build/${APP_NAME}-darwin-386 -ldflags "$(LDFLAGS)"
	@echo -e "$(OK_COLOR)==> Trying to compress binary ...$(NO_COLOR)"
	@upx --brute  build/${APP_NAME}-darwin-386 || upx-ucl --brute  build/${APP_NAME}-darwin-386 || echo -e "$(WARN_COLOR)==> No tools found to compress binary.$(NO_COLOR)"

	@echo -e "$(OK_COLOR)==> Building for darwin64 ...$(NO_COLOR)"
	cd $(FAKE_PACKAGE) && GOPATH=$(FAKE_GOPATH) CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o build/${APP_NAME}-darwin-amd64 -ldflags "$(LDFLAGS)"
	@echo -e "$(OK_COLOR)==> Trying to compress binary ...$(NO_COLOR)"
	@upx --brute  build/${APP_NAME}-darwin-amd64 || upx-ucl --brute  build/${APP_NAME}-darwin-amd64 || echo -e "$(WARN_COLOR)==> No tools found to compress binary.$(NO_COLOR)"

#	@echo -e "$(OK_COLOR)==> Building for win32 ...$(NO_COLOR)"
#	cd $(FAKE_PACKAGE) && GOPATH=$(FAKE_GOPATH) CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -o build/${APP_NAME}-win-386 -ldflags "$(LDFLAGS)"
#	@echo -e "$(OK_COLOR)==> Trying to compress binary ...$(NO_COLOR)"
#	@upx --brute  build/${APP_NAME}-win-386 || upx-ucl --brute  build/${APP_NAME}-win-386 || echo -e "$(WARN_COLOR)==> No tools found to compress binary.$(NO_COLOR)"

#	@echo -e "$(OK_COLOR)==> Building for win64 ...$(NO_COLOR)"
#	cd $(FAKE_PACKAGE) && GOPATH=$(FAKE_GOPATH) CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o build/${APP_NAME}-win-amd64 -ldflags "$(LDFLAGS)"
#	@echo -e "$(OK_COLOR)==> Trying to compress binary ...$(NO_COLOR)"
#	@upx --brute  build/${APP_NAME}-win-amd64 || upx-ucl --brute  build/${APP_NAME}-win-amd64 || echo -e "$(WARN_COLOR)==> No tools found to compress binary.$(NO_COLOR)"

	@echo -e "$(OK_COLOR)==> Archiving ...$(NO_COLOR)"
	@tar -zcvf build/$(ARCHIVE) LICENSE README.md build/$(APP_NAME)-*

clean:
	@if [ -x $(APP_NAME) ]; then rm $(APP_NAME); fi
	@if [ -d build ]; then rm -R build; fi
	@if [ -d $(FAKE_GOPATH) ]; then rm -R $(FAKE_GOPATH); fi
	@rm -rf ./plugin

compress:
	@echo -e "$(OK_COLOR)==> Trying to compress binary ...$(NO_COLOR)"
	@upx --brute $(APP_NAME) || upx-ucl --brute $(APP_NAME) || echo -e "$(WARN_COLOR)==> No tools found to compress binary.$(NO_COLOR)"

format:
	@echo -e "$(OK_COLOR)==> Formatting...$(NO_COLOR)"
	go fmt ./gluster/...

test: test-unit test-integration
	@echo -e "$(OK_COLOR)==> Running test...$(NO_COLOR)"
	gocovmerge coverage.unit.out coverage.inte.out > coverage.all
#	go tool cover -html=coverage.all -o coverage.html

test-unit: dev-deps deps format
	@echo -e "$(OK_COLOR)==> Running unit tests...$(NO_COLOR)"
	go vet ./gluster/... || true
	go test -v -race -coverprofile=coverage.unit.out -covermode=atomic ./gluster/driver

test-integration: dev-deps deps format
	@echo -e "$(OK_COLOR)==> Running integration tests...$(NO_COLOR)"
	go test -v -timeout 1h -race -coverprofile=coverage.inte.out -covermode=atomic -coverpkg ./gluster/driver ./gluster/integration

test-coverage: test
	@echo -e "$(OK_COLOR)==> Uploading coverage ...$(NO_COLOR)"
	curl -s https://codecov.io/bash | bash -s - -F unittests -f coverage.unit.out
	curl -s https://codecov.io/bash | bash -s - -F integration -f coverage.inte.out
#Need CODECOV_TOKEN=:uuid

docs:
	@echo -e "$(OK_COLOR)==> Serving docs at http://localhost:$(DOC_PORT).$(NO_COLOR)"
	@godoc -http=:$(DOC_PORT)

lint: dev-deps
	gometalinter --deadline=5m --concurrency=2 --vendor --disable=gotype --errors ./...
	gometalinter --deadline=5m --concurrency=2 --vendor --disable=gotype ./... || echo "Something could be improved !"
#	gometalinter --deadline=5m --concurrency=2 --vendor ./... # disable gotype temporary

dev-deps:
	@echo -e "$(OK_COLOR)==> Installing developement dependencies...$(NO_COLOR)"
	@go get github.com/nsf/gocode
	@go get github.com/alecthomas/gometalinter
	@go get github.com/golang/dep/cmd/dep #Vendoring
	@go get github.com/wadey/gocovmerge
	@$(GOPATH)/bin/gometalinter --install > /dev/null

update-dev-deps:
	@echo -e "$(OK_COLOR)==> Installing/Updating developement dependencies...$(NO_COLOR)"
	go get -u github.com/nsf/gocode
	go get -u github.com/alecthomas/gometalinter
	go get -u github.com/golang/dep/cmd/dep #Vendoring
	go get -u github.com/wadey/gocovmerge
	$(GOPATH)/bin/gometalinter --install --update

deps:
	@echo -e "$(OK_COLOR)==> Installing dependencies ...$(NO_COLOR)"
	@$(GOPATH)/bin/dep ensure

update-deps: dev-deps
	@echo -e "$(OK_COLOR)==> Updating all dependencies ...$(NO_COLOR)"
	$(GOPATH)/bin/dep ensure -update


done:
	@echo -e "$(OK_COLOR)==> Done.$(NO_COLOR)"

.PHONY: all build compile clean compress format test docs lint dev-deps update-dev-deps deps update-deps done
