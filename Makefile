GO      ?= go
DOTNET  ?= dotnet
PKG     := ./...

.PHONY: all test race checkptr vet fmt lint cover bench fuzz testdata testdata-check cross clean

all: fmt vet test

test:
	$(GO) test $(PKG)

race:
	$(GO) test -race $(PKG)

# Validates the uint32 struct-to-array aliasing the arithmetic core relies on.
checkptr:
	$(GO) test -gcflags=all=-d=checkptr $(PKG)

vet:
	$(GO) vet $(PKG)

fmt:
	@out="$$(gofmt -l . )"; \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

lint:
	golangci-lint run

cover:
	$(GO) test -coverprofile=coverage.out $(PKG)
	$(GO) tool cover -func=coverage.out | tail -1

bench:
	$(GO) test -run=NONE -bench=. -benchmem $(PKG)

fuzz:
	$(GO) test -run=NONE -fuzz=FuzzParse -fuzztime=60s .
	$(GO) test -run=NONE -fuzz=FuzzFormat -fuzztime=60s .
	$(GO) test -run=NONE -fuzz=FuzzArithmetic -fuzztime=60s .

# Regenerates the golden tables from the .NET runtime. Requires a .NET SDK.
# Output is deterministic, so a clean tree after running this is the check that
# the Go implementation and .NET still agree on what the reference answers are.
testdata:
	$(DOTNET) run --project testdata/gen -- testdata

testdata-check: testdata
	@git diff --exit-code testdata/ || \
		{ echo "testdata/ changed; commit the regenerated tables"; exit 1; }

# Every architecture the package claims to support.
cross:
	@for arch in 386 amd64 arm arm64 loong64 mips mips64 mipsle ppc64 ppc64le riscv64 s390x; do \
		printf '%-10s' $$arch; \
		GOOS=linux GOARCH=$$arch $(GO) build $(PKG) && echo ok || exit 1; \
	done
	@printf '%-10s' wasm; GOOS=js GOARCH=wasm $(GO) build $(PKG) && echo ok

clean:
	$(GO) clean -testcache
	rm -f coverage.out
	rm -rf testdata/gen/bin testdata/gen/obj
