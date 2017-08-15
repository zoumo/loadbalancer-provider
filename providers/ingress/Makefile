
all: push

RELEASE?=v0.2.0
GOOS?=linux
PREFIX?=cargo.caicloud.io/caicloud/ingress-controller-sidecar

PKG=github.com/caicloud/loadbalancer-provider/providers/ingress
REPO_INFO=$(shell git config --get remote.origin.url)

ifndef COMMIT
  COMMIT := git-$(shell git rev-parse --short HEAD)
endif

target=ingress-controller-sidecar

test:
	go list ./... | grep -v '/vendor/' | grep -v '/tests/' | xargs go test 

build: clean test
	GOOS=${GOOS} go build -i -v -o ${target} \
	-ldflags "-s -w -X ${PKG}/version.RELEASE=${RELEASE} -X ${PKG}/version.COMMIT=${COMMIT} -X ${PKG}/version.REPO=${REPO_INFO}" \
	${PKG}/cmd

debug: clean
	go build -i -v -o ${target} \
	-ldflags "-s -w -X ${PKG}/version.RELEASE=${RELEASE} -X ${PKG}/version.COMMIT=${COMMIT} -X ${PKG}/version.REPO=${REPO_INFO}" \
	${PKG}/cmd

image: build
	docker build -t $(PREFIX):$(RELEASE) .

push: image
	docker push $(PREFIX):$(RELEASE)

clean:
	rm -f ${target}
