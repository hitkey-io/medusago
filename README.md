# MedusaGo

A Go implementation of the [Medusa](https://github.com/medusajs/medusa) commerce
backend. It is API-compatible with Medusa v2, so the original Medusa Admin
dashboard, `@medusajs/js-sdk` and Medusa storefronts work against it unchanged,
and it runs on the same database as a Node.js Medusa application.

Baseline: Medusa `v2.22.6`.

## Getting started

Requirements: [Go](https://go.dev/dl/) 1.26 or later, [PostgreSQL](https://www.postgresql.org/download/)
and Git. Node.js is only needed for the optional Next.js storefront and for
admin customizations.

Create a project, the same way `create-medusa-app` does for Medusa:

```sh
go run github.com/hitkey-io/medusago/packages/cli/create-medusago-app@latest my-medusa-store
```

> A shorter `npx create-medusago-app my-medusa-store` (prebuilt binaries
> through npm, no Go toolchain needed to run the installer) is in development.

The installer:

1. creates the `my-medusa-store` directory from the
   [MedusaGo starter](https://github.com/hitkey-io/medusago-starter)
   (`apps/backend`, and `apps/storefront` if you choose the Next.js storefront);
2. creates the PostgreSQL database `medusa-my-medusa-store` (it asks for your
   PostgreSQL credentials if the default `postgres` user without a password
   does not work);
3. runs the migrations and creates an admin invite;
4. starts the application at `http://localhost:9000` and opens the admin at
   `http://localhost:9000/app` to create your admin user.

Options are the same as `create-medusa-app`: `--db-url`, `--skip-db`,
`--no-migrations`, `--no-browser`, `--directory-path`, `--repo-url`,
`--with-nextjs-starter`, `--verbose`, `--version` (run with `-h` for the list).

### Working on a project

Run the commands from `apps/backend`. They mirror the Medusa CLI:

| Medusa                               | MedusaGo                           |
| ------------------------------------ | ---------------------------------- |
| `npm run dev`                        | `go run . develop`                 |
| `npx medusa db:setup`                | `go run . db:setup`                |
| `npx medusa db:migrate`              | `go run . db:migrate`              |
| `npx medusa user -e <email> -p <pw>` | `go run . user -e <email> -p <pw>` |
| `npm run build` / `npm run start`    | `go run . build` / `go run . start` |

The admin is served by the backend at `/app`.

### Using an existing Medusa database

MedusaGo reads and writes the database of a Medusa `v2.22.6` application as is,
and Medusa can take over a database MedusaGo has used. Stop the Node.js server
before starting MedusaGo on the same database: workflow executions that are in
progress are not carried over.

## Compatibility

Every route is checked against upstream Medusa: the test suite replays recorded
requests (2,600+ requests in 96 scenarios) against both implementations and
compares status codes, headers, cookies and response bodies byte for byte, and
compares the database contents after each scenario. Module and Query behaviour
is checked the same way.

Feature-flagged areas (index engine, translations, view configurations) are
supported with their flags enabled. Known gaps are tracked in the
[issues](https://github.com/hitkey-io/medusago/issues). Medusa Enterprise
Edition features (RBAC, SSO) are not part of this project.

## Performance

Medusa 2.22.6 (production build, Node.js 24) and MedusaGo v0.1.0 on copies of
the same PostgreSQL 17 database (304 products, 920 variants), without Redis,
on an Apple M4 Pro VM (7 vCPU, 12 GB). Both servers return byte-identical
responses.

| Resources                 | Medusa, 1 process | Medusa, cluster ×6 | MedusaGo |
| ------------------------- | ----------------: | -----------------: | -------: |
| Startup                   |             1.6 s |              3.6 s |    0.1 s |
| Memory, idle              |            426 MB |           2,740 MB |    82 MB |
| Memory, peak under load   |          1,000 MB |           4,100 MB |   205 MB |

Throughput, requests per second (32 concurrent clients; checkout: completed
orders per second, 16 concurrent clients):

| Endpoint                                     | Medusa, 1 process | Medusa, cluster ×6 | MedusaGo | MedusaGo vs cluster |
| -------------------------------------------- | ----------------: | -----------------: | -------: | ------------------: |
| `GET /store/regions`                         |               904 |              1,714 |    6,432 |                3.8× |
| `GET /store/product-categories`              |               669 |              1,619 |    4,632 |                2.9× |
| `GET /store/products`                        |                88 |                187 |      611 |                3.3× |
| `GET /store/products` with calculated prices |                61 |                135 |      387 |                2.9× |
| `GET /store/products/:id` with prices        |               185 |                387 |    1,310 |                3.4× |
| `GET /admin/products`                        |                52 |                135 |      319 |                2.4× |
| `GET /admin/orders`                          |                60 |                174 |      402 |                2.3× |
| Checkout, cart to order (8 requests)         |               3.7 |               11.6 |     47.3 |                4.1× |

Latency, median with a single client:

| Endpoint                                     |  Medusa | MedusaGo |
| -------------------------------------------- | ------: | -------: |
| `GET /store/regions`                         |  1.5 ms |   0.7 ms |
| `GET /store/products`                        | 15.6 ms |   6.5 ms |
| `GET /store/products` with calculated prices | 26.5 ms |  10.1 ms |
| `GET /store/products/:id` with prices        | 10.7 ms |   3.8 ms |
| `GET /admin/products`                        | 27.0 ms |  11.6 ms |
| `GET /admin/orders`                          | 22.3 ms |   9.9 ms |
| Checkout, cart to order (8 requests)         |  385 ms |   114 ms |

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
go build ./...
go test ./...

# Run the server from the repository (PostgreSQL in DATABASE_URL)
go run ./packages/cli/medusa-cli db:setup
go run ./packages/cli/medusa-cli start --port 9000

# Original JavaScript packages (yarn is vendored in .yarn/releases)
node .yarn/releases/yarn-4.18.1.cjs install
node .yarn/releases/yarn-4.18.1.cjs build
```

The compatibility suites in `integration-tests/` need the reference
PostgreSQL and an upstream Medusa checkout to record their golden data; see
`integration-tests/http-diff/record.sh` and
`integration-tests/modules-diff/generate.sh`.

## License

MIT, see [LICENSE](LICENSE), except for the Medusa Enterprise Edition
materials identified in [ENTERPRISE-LICENSE.md](ENTERPRISE-LICENSE.md).
