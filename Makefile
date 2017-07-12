default: check

.PHONY: check ci dependencies

dependencies:
	go get -t -v ./...

check: dependencies
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install --update
	gometalinter -j2 --config "$(CURDIR)/gometalinter.json" ./...

build: dependencies
	go build ./...

ci: build #TODO fix warnings check
	true
