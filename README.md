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

## Development

```sh
go test ./...
go run ./packages/cli/medusa-cli start --port 9000
```

## License

MIT, see [LICENSE](LICENSE). Medusa Enterprise Edition materials (RBAC, SSO)
are not part of this project.
