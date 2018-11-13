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

clean-generate:
	rm -f pkg/apis/nmstate.io/v1/zz_generated.deepcopy.go
	rm -rf pkg/client
