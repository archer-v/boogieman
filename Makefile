PROJECTPATH:=$(shell pwd)
PROJECTNAME:=boogieman
BUILDPATH:=${PROJECTPATH}/src
DISTRPATH:=${PROJECTPATH}/distr
DISTRBUILDPATH:=${PROJECTPATH}/distbuild
BUILDVERSION:=$(shell dd if=/dev/urandom bs=1 count=4 2>/dev/null | hexdump -e '1/1 "%u"')
BUILDTIMESTAMP:=$(shell date +"%Y-%m-%d_%H%M%S")
BUILDIMAGE:=${PROJECTNAME}_${BUILDTIMESTAMP}_${BUILDVERSION}.tar.gz
CURIMAGE:=${PROJECTNAME}.tar.gz

COMMIT:=$(shell git rev-parse --short HEAD)
BRANCH:=$(shell git rev-parse --abbrev-ref HEAD)
TAG:=$(shell git describe --tags |cut -d- -f1)

ifeq (${TAG},)
    TAG:=devel
endif

LDFLAGS = -ldflags "-X main.gitTag=${TAG} -X main.gitCommit=${COMMIT} -X main.gitBranch=${BRANCH} -X main.build=${BUILDVERSION} -X main.buildTimestamp=${BUILDTIMESTAMP}"

.PHONY: help clean dep build install uninstall

.DEFAULT_GOAL := help

help: ## Display this help screen.
	@echo "Makefile available targets:"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  * \033[36m%-15s\033[0m %s\n", $$1, $$2}'

dep: ## Download the dependencies.
	go mod download

build: dep ## Build executable.
	mkdir -p ./build
	CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o build/${PROJECTNAME} ./src
	strip build/${PROJECTNAME}

clean: ## Clean build directory.
	rm -f ./build/${PROJECTNAME}
	rmdir ./build

#golangci-lint and gosec must be installed, see details:
#https://golangci-lint.run/usage/install/#local-installation
#https://github.com/securego/gosec
lint: dep ## Lint the source files
	golangci-lint run --timeout 5m -E golint
	gosec -quiet ./...

test: dep ## Run tests
	go test -race -p 1 -timeout 300s -coverprofile=.test_coverage.txt ./... && \
	go tool cover -func=.test_coverage.txt | tail -n1 | awk '{print "Total test coverage: " $$3}'
	@rm .test_coverage.txt

distro: build ## Create distro package
	mkdir -p $(DISTRBUILDPATH)
	mkdir -p $(DISTRPATH)
	cp build/${PROJECTNAME} $(DISTRBUILDPATH)
	cd $(DISTRBUILDPATH) && tar -c . | gzip > $(DISTRPATH)/$(BUILDIMAGE)
	rm -r $(DISTRBUILDPATH)
	rm -f "$(DISTRPATH)/$(CURIMAGE)"
	cd $(DISTRPATH) && ln -s $(BUILDIMAGE) $(CURIMAGE)
	echo "Distro build was completed, the distributive package was saved to ${DISTRPATH}/${CURIMAGE}"
