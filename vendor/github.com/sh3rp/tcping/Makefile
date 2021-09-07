all: clean mod build install

clean:
	rm -rf include bin target readme.txt

mod:
	go mod tidy

build:
	mkdir target
	go build -o target/tcping cmd/tcping/tcping.go
	sudo setcap cap_net_raw+ep target/tcping
	cp target/tcping $(GOPATH)/bin/tcping

install:
	sudo setcap cap_net_raw+ep $(GOPATH)/bin/tcping

.PHONY: clean deps build install
