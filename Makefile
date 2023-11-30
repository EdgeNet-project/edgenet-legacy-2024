GOCMD=go
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod vendor
LINTER=golangci-lint

# use git describe after the first release
# XXX: for building from tar balls that don't have git meta data we need a fallback
GIT_VERSION:=$(or \
	$(shell git describe --long --tags 2>/dev/null), \
	$(shell printf "0.0.0.r%s.%s" "$(shell git rev-list --count HEAD)" "$(shell git rev-parse --short HEAD)") \
)
.PHONY: build

# This is for github actions, do not run this in a project.
bootstrap:
	mkdir -p ${HOME}/.kube
	cp ./configs/public.cfg ${HOME}/.kube/config
	cp ./configs/smtp_test_template.yaml ./configs/smtp_test.yaml
	cp ./configs/headnode_template.yaml ./configs/headnode.yaml
	cp ./configs/namecheap_template.yaml ./configs/namecheap.yaml

fedmanctl:
	$(GOMOD) install ./cmd/fedmanctl/fedmanctl.go  

sync:
	$(GOCLEAN) --modcache
	$(GOMOD)

test:
	$(GOCLEAN) -testcache
	$(GOTEST) -covermode atomic ./... -v

build:
	docker-compose -f ./build/yamls/docker-compose.yaml build

rebuild: stop clean build start

start: build
	docker-compose -f ./build/yamls/docker-compose.yaml up -d

run:
	docker-compose -f ./build/yamls/docker-compose.yaml up -d

stop:
	docker-compose -f ./build/yamls/docker-compose.yaml down

clean:
	docker-compose -f ./build/yamls/docker-compose.yaml down --rmi all
	$(GOCLEAN)

lint:
	$(LINTER) run
