# OpenAPI Definitions

This package contains Go types generated from the Ampersand OpenAPI spec
(`api/api.yaml`) defined in <https://github.com/amp-labs/openapi>.

## Generated files

- `api.gen.go` — Go types generated from the `components/schemas` in `api/api.yaml`.
- `commit.json` — the openapi commit these types were generated from.

## Automation

These types are kept in sync automatically:

1. A push to `main` in the `openapi` repo that touches `api/**` fires a
   `repository_dispatch` (`gen-openapi-types`) at this repo.
2. The [`gen-openapi-types`](../.github/workflows/gen-openapi-types.yml) workflow
   regenerates `api.gen.go` and opens a PR on an `auto/openapi-*` branch.
3. [`auto-approve-openapi`](../.github/workflows/auto-approve-openapi.yml)
   approves that PR when the diff is limited to the generated files.

## Regenerating manually

Install oapi-codegen if you haven't:

```shell
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.7.1
```

Regenerate against the openapi `main` branch:

```shell
make gen/main
```

Or against a specific commit:

```shell
make gen OPENAPI_COMMIT_ID=<commit-hash>
```

The api.yaml spec uses cross-file `$ref`s, so generation reads the fully
dereferenced `api/generated/api.json` that the openapi repo publishes (rather
than `api/api.yaml` directly).
