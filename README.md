# MedusaGo

A Go implementation of the [Medusa](https://github.com/medusajs/medusa) commerce
backend, API-compatible with Medusa v2 so the original Medusa Admin dashboard,
`@medusajs/js-sdk` and storefronts work against it unchanged.

Baseline: Medusa `v2.22.4`.

## Layout

The repository mirrors the Medusa monorepo. Differences come only from Go vs
JavaScript:

| Medusa (TypeScript)                 | MedusaGo                            |
| ----------------------------------- | ----------------------------------- |
| `route.ts`, `error-handler.ts`      | `route.go`, `error-handler.go`      |
| `api/admin/products/[id]/route.ts`  | `api/admin/products/id_/route.go`   |
| `__tests__/*.spec.ts`               | `*_test.go` next to the code        |
| `package.json` per package          | a single `go.mod` at the root       |

Route parameter directories use a trailing underscore (`[variant_id]` ->
`variant_id_`) because brackets are not valid in Go import paths.

The original Medusa JavaScript packages (`packages/admin/*`,
`packages/design-system/*`, `packages/core/js-sdk`, `packages/core/types`,
`packages/eslint-plugin`) and the root JavaScript workspace files are copied
1:1 from upstream with `scripts/sync-upstream.sh`. `turbo.json` is adapted
because `@medusajs/medusa` is implemented in Go here.

## Development

```sh
# Go backend
go test ./...
go run ./packages/cli/medusa-cli start --port 9000

# Original JavaScript packages
yarn install
yarn build
```

## License

MIT, see [LICENSE](LICENSE), except for the Medusa Enterprise Edition
materials identified in [ENTERPRISE-LICENSE.md](ENTERPRISE-LICENSE.md).
