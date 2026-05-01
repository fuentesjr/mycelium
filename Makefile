.PHONY: build test dist clean

VERSION ?= v0.3.0
DIST    := dist
CMD     := cmd/mycelium

build:
	cd $(CMD) && go build -o mycelium .

test:
	cd $(CMD) && go test -count=1 ./...

dist: clean
	mkdir -p $(DIST)
	cd $(CMD) && GOOS=darwin GOARCH=amd64 go build -o $(CURDIR)/$(DIST)/mycelium-$(VERSION)-darwin-amd64 .
	cd $(CMD) && GOOS=darwin GOARCH=arm64 go build -o $(CURDIR)/$(DIST)/mycelium-$(VERSION)-darwin-arm64 .
	cd $(CMD) && GOOS=linux  GOARCH=amd64 go build -o $(CURDIR)/$(DIST)/mycelium-$(VERSION)-linux-amd64 .
	cd $(CMD) && GOOS=linux  GOARCH=arm64 go build -o $(CURDIR)/$(DIST)/mycelium-$(VERSION)-linux-arm64 .
	cd $(DIST) && for f in mycelium-$(VERSION)-*; do tar -czf $$f.tar.gz $$f && rm $$f; done
	@ls -lh $(DIST)

clean:
	rm -rf $(DIST)
	rm -f $(CMD)/mycelium
