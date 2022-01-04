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

sync:
	$(GOCLEAN) --modcache
	$(GOMOD)

all:
	GO111MODULE=on GOBIN=${GOPATH}/bin go install -mod=vendor \
		-gcflags="all=-trimpath=$GOPATH" \
		-asmflags="all=-trimpath=$GOPATH" \
		-ldflags="-X github.com/EdgeNet-Project/edgenet.CurrentVersion=$(GIT_VERSION)" \
		./cmd/...

bootstrap:
	mkdir ${HOME}/.kube
	cp ./configs/public.cfg ${HOME}/.kube/config
	cp ./configs/smtp_test_template.yaml ./configs/smtp_test.yaml
	cp ./configs/headnode_template.yaml ./configs/headnode.yaml
	cp ./configs/namecheap_template.yaml ./configs/namecheap.yaml

test:
	$(GOCLEAN) -testcache ./...
	$(GOTEST) -covermode atomic ./... -v
	find ./assets/certs ! -name 'README.md' -type f -exec rm -f {} +
	find ./assets/kubeconfigs ! -name 'README.md' -type f -exec rm -f {} +

build:
	docker-compose -f ./build/yamls/docker-compose.yml build

rebuild: stop clean build start

start: build
	docker-compose -f ./build/yamls/docker-compose.yml up -d

run:
	docker-compose -f ./build/yamls/docker-compose.yml up -d

stop:
	docker-compose -f ./build/yamls/docker-compose.yml down

clean:
	docker-compose -f ./build/yamls/docker-compose.yml down --rmi all
	$(GOCLEAN)

lint:
	$(LINTER) run
