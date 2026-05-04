.PHONY: build test dist clean npm-dist npm-publish

VERSION ?= v0.1.5
DIST    := dist
CMD     := cmd/mycelium
NPM_DIR := $(DIST)/npm

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

npm-dist: dist
	@for plat in darwin-arm64 darwin-amd64 linux-arm64 linux-amd64; do \
	  os=$${plat%-*}; goarch=$${plat#*-}; \
	  case $$goarch in \
	    amd64) nodearch=x64;; \
	    arm64) nodearch=arm64;; \
	    *) echo "unknown arch $$goarch"; exit 1;; \
	  esac; \
	  pkg=$(NPM_DIR)/cli-$$plat; \
	  mkdir -p $$pkg; \
	  tar -xzf $(DIST)/mycelium-$(VERSION)-$$plat.tar.gz -C $$pkg --strip-components=0; \
	  mv $$pkg/mycelium-$(VERSION)-$$plat $$pkg/mycelium; \
	  chmod +x $$pkg/mycelium; \
	  ver=$$(echo $(VERSION) | sed 's/^v//'); \
	  printf '{\n  "name": "@fuentesjr/mycelium-cli-%s",\n  "version": "%s",\n  "description": "Mycelium CLI binary for %s",\n  "license": "MIT",\n  "os": ["%s"],\n  "cpu": ["%s"],\n  "files": ["mycelium"],\n  "repository": "https://github.com/fuentesjr/mycelium"\n}\n' \
	    $$plat $$ver $$plat $$os $$nodearch > $$pkg/package.json; \
	done
	@ls -lh $(NPM_DIR)

npm-publish: npm-dist
	@for plat in darwin-arm64 darwin-amd64 linux-arm64 linux-amd64; do \
	  cd $(NPM_DIR)/cli-$$plat && npm publish --access=public && cd $(CURDIR); \
	done
	cd extensions/pi-mycelium && npm publish --access=public

clean:
	rm -rf $(DIST)
	rm -f $(CMD)/mycelium
