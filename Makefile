HOSTNAME=registry.terraform.io
NAMESPACE=local
NAME=litellm
VERSION=1.0.0
OS_ARCH=darwin_amd64

default: install

build:
	go build -o terraform-provider-${NAME}

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv terraform-provider-${NAME} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}/terraform-provider-${NAME}_v${VERSION}

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run

clean:
	rm -f terraform-provider-${NAME}
	rm -rf ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}

# Start LiteLLM + DB for local/smoke testing. Run once before make smoke.
local:
	cd internal_testing && docker compose up -d
	@echo "Run make logs to follow LiteLLM logs, then make smoke resources=... or datasources=..."

# Follow LiteLLM proxy logs (run after make local).
logs:
	cd internal_testing && docker compose logs -f litellm

# Fetch the live OpenAPI spec from the running LiteLLM proxy (FastAPI serves it
# at /openapi.json) and pretty-print it to openapi.json — for reference when
# adding/changing resource fields. This is NOT a build input; it is generated,
# gitignored, and always reflects the LiteLLM version the container is running,
# so it can't silently drift the way a committed snapshot does. Requires `make
# local`. Override the endpoint with LITELLM_URL.
LITELLM_URL ?= http://localhost:4000
openapi:
	@curl -fsS "$(LITELLM_URL)/openapi.json" | python3 -m json.tool > openapi.json
	@printf 'Wrote openapi.json (LiteLLM %s)\n' "$$(python3 -c 'import json;print(json.load(open("openapi.json"))["info"]["version"])')"

# Smoke test: for each given file run plan -> apply -> destroy in .smoke (one file at a time).
# Requires: make local (LiteLLM + DB up), make build. At least one of resources= or datasources= is required (comma-separated).
# Usage:
#   make smoke resources=model_minimal.tf
#   make smoke resources=model_minimal.tf,key_minimal.tf
#   make smoke datasources=keys_list.tf
#   make smoke resources=model_minimal.tf datasources=keys_list.tf
# CURDIR is Make's current working directory (repo root); passed so the script finds internal_testing and the provider binary.
smoke: build
	@test -f terraform-provider-$(NAME) || (echo "Run 'make build' first."; exit 1)
	@test -n "$(resources)$(datasources)" || (echo "Usage: make smoke resources=file.tf [datasources=file.tf]"; exit 1)
	@sh internal_testing/smoke.sh $(CURDIR) resources $(strip $(subst ,, ,$(resources))) datasources $(strip $(subst ,, ,$(datasources)))

.PHONY: build install test fmt vet lint clean local logs openapi smoke
