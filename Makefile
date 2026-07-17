.PHONY: build test dist clean npm-dist npm-publish

VERSION ?= v0.5.0
DIST    := dist
CMD     := cmd/mycelium
NPM_DIR := $(DIST)/npm

build:
	go build -o $(CMD)/mycelium ./$(CMD)

test:
	go test -count=1 ./...
	npm test --prefix extensions/pi-mycelium

dist: clean
	mkdir -p $(DIST)
	GOOS=darwin GOARCH=amd64 go build -o $(CURDIR)/$(DIST)/mycelium-$(VERSION)-darwin-amd64 ./$(CMD)
	GOOS=darwin GOARCH=arm64 go build -o $(CURDIR)/$(DIST)/mycelium-$(VERSION)-darwin-arm64 ./$(CMD)
	GOOS=linux  GOARCH=amd64 go build -o $(CURDIR)/$(DIST)/mycelium-$(VERSION)-linux-amd64 ./$(CMD)
	GOOS=linux  GOARCH=arm64 go build -o $(CURDIR)/$(DIST)/mycelium-$(VERSION)-linux-arm64 ./$(CMD)
	# shellcheck disable=SC1073,SC1061,SC1036,SC1062,SC1072
	cd $(DIST) && for f in mycelium-$(VERSION)-*; do tar -czf $$f.tar.gz $$f && rm $$f; done
	ls -lh $(DIST)

npm-dist: dist
	# shellcheck disable=SC1073,SC1061,SC1036,SC1062,SC1072
	for plat in darwin-arm64 darwin-amd64 linux-arm64 linux-amd64; do \
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
	  node -e 'const fs = require("fs"); const [plat, ver, os, cpu, file] = process.argv.slice(1); fs.writeFileSync(file, JSON.stringify({ name: "@fuentesjr/mycelium-cli-" + plat, version: ver, description: "Mycelium CLI binary for " + plat, license: "MIT", os: [os], cpu: [cpu], files: ["mycelium"], repository: "https://github.com/fuentesjr/mycelium" }, null, 2) + "\n");' $$plat $$ver $$os $$nodearch $$pkg/package.json; \
	done
	ls -lh $(NPM_DIR)

npm-publish: npm-dist
	# shellcheck disable=SC1073,SC1061,SC1036,SC1062,SC1072
	for plat in darwin-arm64 darwin-amd64 linux-arm64 linux-amd64; do \
	  cd $(NPM_DIR)/cli-$$plat && npm publish --access=public --provenance && cd $(CURDIR); \
	done
	cd extensions/pi-mycelium && npm publish --access=public --provenance

clean:
	rm -rf $(DIST)
	rm -f $(CMD)/mycelium
