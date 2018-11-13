all: build

build:
	cd cmd/client && go build

generate:
	hack/update-codegen.sh

test:
	hack/test.sh

dep:
	dep ensure -v

clean-dep:
	rm -f ./Gopkg.lock
	rm -rf ./vendor

