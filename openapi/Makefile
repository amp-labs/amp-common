# Generate Go types from the Ampersand OpenAPI spec (api/api.yaml).
#
# The api.yaml spec uses cross-file $refs, so we generate from the fully
# dereferenced JSON that the openapi repo publishes at api/generated/api.json.

ifneq ($(MAKECMDGOALS),gen/main)
ifndef OPENAPI_COMMIT_ID
OPENAPI_COMMIT_ID := $(shell read -p "Enter OPENAPI_COMMIT_ID (commit hash, default 'main'): " id < /dev/tty; echo $${id:-main})
endif
endif
API_JSON = https://raw.githubusercontent.com/amp-labs/openapi/$(OPENAPI_COMMIT_ID)/api/generated/api.json

.PHONY: ensure-oapi-codegen
ensure-oapi-codegen:
	@command -v oapi-codegen >/dev/null 2>&1 || { \
		echo "oapi-codegen not found, installing..."; \
		go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.7.1; \
	}

# Regenerate the Go types for the main branch of the openapi repository.
.PHONY: gen/main
gen/main:
	$(MAKE) gen OPENAPI_COMMIT_ID=$$(git ls-remote https://github.com/amp-labs/openapi.git refs/heads/main | cut -f1)

# Regenerate the Go types from a specific commit of the openapi repository.
.PHONY: gen
gen: ensure-oapi-codegen gen/api
	echo "{\"commit_id\": \"${OPENAPI_COMMIT_ID}\"}" > commit.json

.PHONY: gen/api
gen/api:
	oapi-codegen --config=oapi.config.yaml --o=api.gen.go $(API_JSON)

# Assumes that openapi is checked out in the same parent directory (amp-labs)
# as this repo, and that its api/generated/api.json is up to date
# (run `pnpm run gen:json` in the openapi repo to refresh it).
.PHONY: gen/local
gen/local: ensure-oapi-codegen
	oapi-codegen --config=oapi.config.yaml --o=api.gen.go ../../openapi/api/generated/api.json
