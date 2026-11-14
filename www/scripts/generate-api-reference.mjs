// Generates the REST API reference (content/api/**) and its sidebar
// (src/generated/api-sidebar.mjs) from the upstream OpenAPI specs.
//
// Input:  reference/medusa/www/apps/api-reference/specs/{admin,store}/openapi.full.yaml
//         (the bundled spec: tags, paths, schemas and code samples in one file)
// Output: content/api/page.mdx, content/api/{admin,store}/page.mdx (introduction)
//         and content/api/<area>/<tag>/page.mdx (one page per tag).
//
// An operation is published only when the Go server registers its method and
// path: the registered routes come from `go run ./www/scripts/api-routes` (a
// snapshot, scripts/api-routes.snapshot.tsv, is used when Go is not available).
// Enterprise operations (roles, policies, SSO) are always dropped.
//
// The output is deterministic: rerun this script whenever the specs, the routes
// or the Go server change. Never edit content/api by hand.
import fs from "node:fs"
import path from "node:path"
import { spawnSync } from "node:child_process"
import YAML from "yaml"

const root = path.resolve(import.meta.dirname, "..")
const repo = path.resolve(root, "..")
const specsDir = path.join(repo, "reference/medusa/www/apps/api-reference/specs")
// The generated pages are committed; without the upstream clone (CI) they are used as they are.
if (!fs.existsSync(path.join(specsDir, "admin", "openapi.full.yaml"))) {
  console.log("upstream specs not found: keeping the committed API reference pages")
  process.exit(0)
}
const outDir = path.join(root, "content/api")
const snapshotFile = path.join(import.meta.dirname, "api-routes.snapshot.tsv")

const AREAS = [
  { key: "admin", title: "Admin API", sidebarTitle: "Admin API" },
  { key: "store", title: "Store API", sidebarTitle: "Store API" },
]

// Limits of the nested type lists (a node at depth N shows children when N < max).
// A deeper level is used while the list stays under NODE_BUDGET entries.
const MAX_DEPTH_REQUEST = 5
const MAX_DEPTH_RESPONSE = 4
const MIN_DEPTH = 2
const NODE_BUDGET = 120
const ENUM_INLINE_MAX = 8

// ---------------------------------------------------------------------------
// Registered routes
// ---------------------------------------------------------------------------

function loadGoRoutes() {
  const res = spawnSync("go", ["run", "./www/scripts/api-routes"], { cwd: repo, encoding: "utf8", maxBuffer: 64 * 1024 * 1024 })
  if (res.status === 0 && res.stdout.trim()) {
    fs.writeFileSync(snapshotFile, res.stdout)
    return res.stdout
  }
  if (fs.existsSync(snapshotFile)) {
    console.warn("api reference: `go run` failed, using scripts/api-routes.snapshot.tsv")
    return fs.readFileSync(snapshotFile, "utf8")
  }
  throw new Error("api reference: cannot list the registered routes: " + (res.stderr || res.error))
}

const goRoutes = loadGoRoutes()
  .trim()
  .split("\n")
  .map((line) => {
    const [method, p, flag] = line.split("\t")
    return { method, segments: p.split("/").filter(Boolean), flag: flag || "" }
  })

// A Go parameter segment (":id") matches a spec parameter ("{id}"), and a literal segment
// must be equal. The only exception are the auth routes, which Go registers once for
// every actor type (":actor_type") while the spec documents them for "user" and "customer".
function findGoRoute(method, specPath) {
  const segs = specPath.split("/").filter(Boolean)
  const isParam = (s) => s.startsWith("{")
  return goRoutes.find(
    (r) =>
      r.method === method &&
      r.segments.length === segs.length &&
      r.segments.every((g, i) => (g.startsWith(":") ? isParam(segs[i]) || g === ":actor_type" : g === segs[i]))
  )
}

// Enterprise operations (RBAC roles/policies and SSO): never published.
function isEnterprise(specPath, op) {
  if (/(^|\/)(roles?|rbac|policies|policy)(\/|$)/i.test(specPath)) return true
  // POST /auth/{auth_provider}/user (SSO user creation), not /auth/user/{auth_provider}
  if (/^\/auth\/\{auth_provider\}\/user$/.test(specPath)) return true
  return false
}

// ---------------------------------------------------------------------------
// Text helpers
// ---------------------------------------------------------------------------

const slugify = (s) =>
  s
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, "")
    .trim()
    .replace(/\s+/g, "-")

const siteHas = (p) => fs.existsSync(path.join(root, "content", p.replace(/^\//, ""), "page.mdx"))
// A link to a page of this site when it exists, the bare text otherwise.
const siteLink = (text, p) => (siteHas(p) ? `[${text}](${p})` : text)

// Anchors of the introduction pages that descriptions may link to.
const LOCAL_ANCHORS = new Set(["filter-operators", "manage-metadata", "select-fields-and-relations", "pagination", "authentication", "query-parameter-types", "errors"])

function rewriteLinks(text, area) {
  return text.replace(/\[([^\]]*)\]\(([^)\s]+)\)/g, (all, label, url) => {
    const m = /^(?:https:\/\/docs\.medusajs\.com)?\/api\/(admin|store)(?:#([a-z0-9-]+))?$/.exec(url) || /^#([a-z0-9-]+)$/.exec(url)
    if (m) {
      const target = m.length === 3 ? m[1] : area
      const anchor = m.length === 3 ? m[2] : m[1]
      if (!anchor) return `[${label}](/api/${target})`
      return LOCAL_ANCHORS.has(anchor) ? `[${label}](/api/${target}#${anchor})` : label
    }
    if (/^https?:\/\/docs\.medusajs\.com/.test(url) || /^\/(learn|resources|api|user-guide)/.test(url)) return label
    return all
  })
}

// Removes the mentions of Enterprise-only features from upstream text and samples.
function scrub(text) {
  return text
    .replace(/, such as the RBAC Module, is installed/g, " is installed")
    .replace(/such as the RBAC Module, is installed/g, "is installed")
    .replace(/isRbacInstalled/g, "isInstalled")
    .replace(/module\.key === "rbac"/g, 'module.key === "brand"')
    .replace(/such as `rbac` or a custom module's name/g, "such as the name of a custom module")
    .replace(/`oidc` or `emailpass`/g, "`emailpass`")
    .replace(/an\s+OIDC identity provider/g, "an external identity provider")
    .replace(/are only available for Cloud users using the \[Loyalty Plugin\]\([^)]*\)/g, "are only available when the loyalty plugin is installed")
}

function sanitize(text, area) {
  if (!text) return ""
  let t = scrub(String(text).trim())
  t = t.replace(/<\/?Note[^>]*>/g, "")
  t = rewriteLinks(t, area)
  t = t.replace(/\bMedusa (application|server|backend|project|instance)(s?)\b/gi, (_, w, s) => `MedusaGo ${w.toLowerCase()}${s}`)
  return t.replace(/\n{3,}/g, "\n\n").trim()
}

// Escapes what MDX reads as syntax ({, }, <) outside code spans and fences.
function escapeMdx(text) {
  return text
    .split(/(```[\s\S]*?```|`[^`\n]*`)/g)
    .map((part, i) => (i % 2 === 1 ? part : part.replace(/[{}<]/g, (c) => `\\${c}`)))
    .join("")
}

const mdxText = (text, area) => escapeMdx(sanitize(text, area))

// ---------------------------------------------------------------------------
// Schema -> TypeList types
// ---------------------------------------------------------------------------

function makeSchemaTools(spec, area) {
  const schemas = spec.components.schemas
  const deref = (ref) => {
    const name = ref.replace("#/components/schemas/", "")
    return { name, schema: schemas[name] }
  }

  const FLAG_KEYS = { index: "index_engine", index_engine: "index_engine", translation: "translation", view_configurations: "view_configurations" }

  const quote = (v) => (typeof v === "string" ? JSON.stringify(v) : String(v))

  // analyze returns { parts, description, children, defaultValue, example, featureFlag, deprecated }
  function analyze(node, depth, maxDepth, stack) {
    if (!node || typeof node !== "object") return { parts: ["any"], children: [] }
    if (node.$ref) {
      const { name, schema } = deref(node.$ref)
      if (!schema) return { parts: ["object"], children: [] }
      if (stack.includes(name)) {
        const r = analyze({ ...schema, properties: undefined, allOf: undefined, oneOf: undefined, anyOf: undefined }, depth, maxDepth, stack)
        return { ...r, parts: ["object"], children: [] }
      }
      return analyze(schema, depth, maxDepth, [...stack, name])
    }

    const own = {
      description: node.description,
      defaultValue: node.default !== undefined ? (typeof node.default === "string" ? JSON.stringify(node.default) : JSON.stringify(node.default)) : undefined,
      example: ["string", "number", "boolean"].includes(typeof node.example) ? String(node.example) : undefined,
      featureFlag: node["x-featureFlag"] ? FLAG_KEYS[node["x-featureFlag"]] || node["x-featureFlag"] : undefined,
      deprecated: node.deprecated === true,
    }

    if (node.allOf) {
      const parts = node.allOf.map((n) => analyze(n, depth, maxDepth, stack))
      const props = []
      for (const p of parts) for (const c of p.children) if (!props.some((x) => x.name === c.name)) props.push(c)
      return {
        ...own,
        parts: ["object"],
        description: own.description || parts.map((p) => p.description).find(Boolean),
        children: props,
      }
    }

    const variants = node.oneOf || node.anyOf
    if (variants) {
      const analyzed = variants.map((v) => analyze(v, depth, maxDepth, stack))
      const parts = []
      for (const a of analyzed) for (const p of a.parts) if (!parts.includes(p)) parts.push(p)
      const merged = new Map()
      for (const a of analyzed) {
        for (const c of a.children) {
          const prev = merged.get(c.name)
          if (!prev) merged.set(c.name, { child: c, count: 1, optional: c.optional })
          else {
            prev.count += 1
            prev.optional = prev.optional || c.optional
          }
        }
      }
      const withChildren = analyzed.filter((a) => a.children.length).length
      const children = [...merged.values()].map(({ child, count, optional }) => ({
        ...child,
        optional: optional || count < withChildren,
      }))
      return {
        ...own,
        parts,
        description: own.description || analyzed.map((a) => a.description).find(Boolean),
        children,
        featureFlag: own.featureFlag || analyzed.map((a) => a.featureFlag).find(Boolean),
      }
    }

    if (node.enum && node.enum.length) {
      if (node.enum.length <= ENUM_INLINE_MAX) return { ...own, parts: node.enum.map(quote), children: [] }
      const list = node.enum.map((v) => `\`${v}\``).join(", ")
      return {
        ...own,
        parts: [node.type === "number" || node.type === "integer" ? "number" : "string"],
        description: `${own.description ? own.description.trim() + " " : ""}Possible values: ${list}.`,
        children: [],
      }
    }

    const type = node.type || (node.properties || node.additionalProperties ? "object" : undefined)

    if (type === "array") {
      const items = analyze(node.items || {}, depth, maxDepth, stack)
      return {
        ...own,
        parts: items.parts.map((p) => (p.includes(" ") || p.startsWith('"') ? `(${p})[]` : `${p}[]`)),
        description: own.description || items.description,
        children: items.children,
        featureFlag: own.featureFlag || items.featureFlag,
      }
    }

    if (type === "object") {
      const children = []
      if (depth < maxDepth) {
        const required = new Set(node.required || [])
        for (const [name, prop] of Object.entries(node.properties || {})) {
          if (isEnterpriseProperty(name)) continue
          children.push(toType(name, prop, !required.has(name), depth + 1, maxDepth, stack))
        }
        if (node.additionalProperties && typeof node.additionalProperties === "object") {
          children.push(toType("[key: string]", node.additionalProperties, false, depth + 1, maxDepth, stack))
        }
      }
      return { ...own, parts: ["object"], children }
    }

    if (type === "integer") return { ...own, parts: ["number"], children: [] }
    return { ...own, parts: [type || "any"], children: [] }
  }

  function toType(name, node, optional, depth, maxDepth, stack, descriptionOverride) {
    const a = analyze(node, depth, maxDepth, stack)
    const description = [sanitize(descriptionOverride || a.description || (node.$ref ? deref(node.$ref).schema?.description : ""), area)]
    const ext = node.externalDocs?.url
    if (ext) {
      const m = /\/api\/(admin|store)#([a-z0-9-]+)$/.exec(ext)
      if (m && LOCAL_ANCHORS.has(m[2])) {
        const label = m[2] === "manage-metadata" ? "Manage Metadata" : "Learn more"
        description.push(`Learn more in the [${label}](/api/${m[1]}#${m[2]}) section.`)
      }
    }
    const t = {
      name,
      type: a.parts.map((p) => `\`${p}\``).join(" \\| "),
      optional,
    }
    const desc = description.filter(Boolean).join(" ")
    if (desc) t.description = desc
    if (a.defaultValue !== undefined) t.defaultValue = a.defaultValue
    if (a.example !== undefined && a.example.length <= 80) t.example = a.example
    if (a.featureFlag) t.featureFlag = a.featureFlag
    if (a.deprecated) t.deprecated = { is_deprecated: true }
    // The filter operators ($eq, $in, ...) are documented once in the introduction.
    const names = new Set(a.children.map((c) => c.name))
    if (names.has("$eq") && names.has("$in") && names.has("$ne")) {
      const rest = a.children.filter((c) => !c.name.startsWith("$"))
      t.description = `${t.description ? t.description + " " : ""}Accepts the [filter operators](/api/${area}#filter-operators).`
      if (rest.length) t.children = rest
    } else if (a.children.length) t.children = a.children
    return t
  }

  const countNodes = (types) => types.reduce((n, t) => n + 1 + countNodes(t.children || []), 0)

  // The top-level fields of an object schema (or a single entry for other types),
  // expanded as deep as the node budget allows.
  function topLevel(schema, maxDepth) {
    let result = []
    for (let depth = maxDepth; depth >= MIN_DEPTH; depth--) {
      const a = analyze(schema, 0, depth, [])
      result = a.parts.length === 1 && a.parts[0] === "object" ? a.children : [toType("(body)", schema, false, 0, depth, [])]
      if (countNodes(result) <= NODE_BUDGET) break
    }
    return result
  }

  // A parameter, expanded as deep as the node budget allows.
  function paramType(name, schema, optional, description) {
    let t
    for (let depth = MAX_DEPTH_REQUEST; depth >= MIN_DEPTH; depth--) {
      t = toType(name, schema, optional, 1, depth, [], description)
      if (countNodes([t]) <= NODE_BUDGET) break
    }
    return t
  }

  return { topLevel, paramType, deref }
}

const ENTERPRISE_PROP = /(^|_)(rbac|policies|policy|roles?)(_|$)/i
function isEnterpriseProperty(name) {
  return ENTERPRISE_PROP.test(name)
}

// Serialises types as a JS literal (smaller than JSON).
function literal(value, indent = 0) {
  const pad = "  ".repeat(indent)
  if (Array.isArray(value)) {
    if (!value.length) return "[]"
    return `[\n${value.map((v) => `${pad}  ${literal(v, indent + 1)}`).join(",\n")}\n${pad}]`
  }
  if (value && typeof value === "object") {
    const entries = Object.entries(value).filter(([, v]) => v !== undefined)
    return `{ ${entries.map(([k, v]) => `${k}: ${literal(v, indent + 1)}`).join(", ")} }`
  }
  return JSON.stringify(value)
}

const typeList = (types) => `<TypeList\n  types={${literal(types, 1)}}\n/>`

// ---------------------------------------------------------------------------
// Operation rendering
// ---------------------------------------------------------------------------

const AUTH_LABELS = {
  api_token: "secret API key (HTTP Basic authorization)",
  jwt_token: "JWT bearer token",
  cookie_auth: "cookie session",
  reset_password: "reset password token",
}

function fence(lang, code) {
  const fenceStr = code.includes("```") ? "````" : "```"
  return `${fenceStr}${lang}\n${code.replace(/\s+$/, "")}\n${fenceStr}`
}

function renderCodeSamples(op) {
  const samples = (op["x-codeSamples"] || []).filter((s) => s.source)
  if (!samples.length) return ""
  const tabs = samples.map((s) => {
    const isShell = /shell|bash/i.test(s.lang)
    const code = scrub(isShell ? s.source.replaceAll("{backend_url}", "http://localhost:9000") : s.source)
    const value = isShell ? "curl" : "js-sdk"
    const label = isShell ? "cURL" : s.label || "JS SDK"
    return `  <CodeTab label=${JSON.stringify(label)} value=${JSON.stringify(value)}>\n\n${fence(isShell ? "bash" : "ts", code)}\n\n  </CodeTab>`
  })
  return `<CodeTabs group="api-sample">\n${tabs.join("\n")}\n</CodeTabs>`
}

function flagEnv(key) {
  return `MEDUSA_FF_${key.toUpperCase()}`
}

function renderOperation(ctx, entry) {
  const { method, specPath, op, goFlag } = entry
  const { area, tools, spec } = ctx
  const out = []
  const heading = `${method} ${specPath}`.replace(/[{}]/g, (c) => `\\${c}`)
  out.push(`## ${heading}`)

  if (op.summary) out.push(`**${escapeMdx(sanitize(op.summary, area))}**`)
  if (op.description && sanitize(op.description, area) !== sanitize(op.summary || "", area)) out.push(mdxText(op.description, area))

  const notes = []
  if (op.deprecated) {
    const msg = op["x-deprecated_message"] ? ` ${sanitize(op["x-deprecated_message"], area)}` : ""
    notes.push(`<Note type="warning" title="Deprecated">\n\nThis API route is deprecated.${escapeMdx(msg)}\n\n</Note>`)
  }
  const flags = new Set()
  for (const f of [op["x-featureFlag"], goFlag]) {
    if (f) flags.add({ index: "index_engine" }[f] || f)
  }
  for (const f of flags) {
    notes.push(
      `<Note title="Feature flag">\n\nThis API route is only available when the \`${f}\` feature flag is enabled. Set the \`${flagEnv(f)}\` environment variable to \`true\` to enable it.\n\n</Note>`
    )
  }
  for (const badge of op["x-badges"] || []) {
    if (/plugin/i.test(badge.text)) {
      notes.push(`<Note title="Plugin">\n\nThis API route is only available when the loyalty plugin is installed in the MedusaGo application.\n\n</Note>`)
      break
    }
  }
  out.push(...notes)

  // authentication
  const schemes = [...new Set((op.security || []).flatMap((s) => Object.keys(s)))].map((k) => AUTH_LABELS[k]).filter(Boolean)
  if (op["x-authenticated"] === false || !schemes.length) out.push("**Authentication:** none.")
  else out.push(`**Authentication:** ${schemes.join(", ")}.`)

  // parameters
  const params = (op.parameters || []).map((p) => (p.$ref ? p : p))
  for (const [where, label] of [
    ["path", "Path parameters"],
    ["header", "Header parameters"],
    ["query", "Query parameters"],
  ]) {
    const list = params.filter((p) => p.in === where)
    if (!list.length) continue
    const types = list.map((p) => {
      return tools.paramType(p.name, p.schema || {}, !(p.required || where === "path"), p.description)
    })
    out.push(`**${label}**`, typeList(types))
  }

  // request body
  const body = op.requestBody?.content?.["application/json"]?.schema
  if (body) {
    const types = tools.topLevel(body, MAX_DEPTH_REQUEST)
    out.push(`**Request body** (\`application/json\`)`)
    if (types.length) out.push(typeList(types))
    else out.push("The request body is an empty JSON object.")
  }

  // examples
  const samples = renderCodeSamples(op)
  if (samples) out.push("**Request examples**", samples)

  // responses
  const success = Object.entries(op.responses || {}).filter(([code]) => /^2\d\d$/.test(code))
  const errorCodes = Object.keys(op.responses || {}).filter((c) => !/^2\d\d$/.test(c))
  out.push("**Responses**")
  for (const [code, res] of success) {
    const content = res.content || {}
    const desc = res.description ? escapeMdx(sanitize(res.description, area)) : ""
    const json = content["application/json"]?.schema
    if (json) {
      const types = tools.topLevel(json, MAX_DEPTH_RESPONSE)
      out.push(`\`${code}\` ${desc}`)
      if (types.length) out.push(typeList(types))
    } else {
      const other = Object.keys(content)[0]
      out.push(`\`${code}\` ${desc.replace(/\.$/, "")}${other ? `. The response body has the content type \`${other}\`.` : ""}`)
    }
  }
  if (errorCodes.length) {
    out.push(`${errorCodes.map((c) => `\`${c}\``).join(", ")}: an [error response](/api/${area}#errors) is returned.`)
  }

  const events = (op["x-events"] || []).filter((e) => e && e.name && !e.deprecated)
  if (events.length) {
    out.push("**Emitted events**")
    out.push(events.map((e) => `- \`${e.name}\`${e.description ? `: ${escapeMdx(sanitize(e.description, area)).replace(/\s*\n\s*/g, " ")}` : ""}`).join("\n"))
  }

  return out.join("\n\n")
}

// ---------------------------------------------------------------------------
// Introduction pages
// ---------------------------------------------------------------------------

const FILTER_OPERATORS = [
  ["$and", "joins query parameters with an AND condition. Each object has the same shape as the expected query parameters."],
  ["$or", "joins query parameters with an OR condition. Each object has the same shape as the expected query parameters."],
  ["$eq", "filters by an exact match."],
  ["$ne", "filters by values not equal to the parameter."],
  ["$in", "filters by values in the array."],
  ["$nin", "filters by values not in the array."],
  ["$not", "filters by values not matching the conditions in the parameter."],
  ["$gt", "filters by values greater than the parameter. Useful for numbers and dates only."],
  ["$gte", "filters by values greater than or equal to the parameter. Useful for numbers and dates only."],
  ["$lt", "filters by values less than the parameter. Useful for numbers and dates only."],
  ["$lte", "filters by values less than or equal to the parameter. Useful for numbers and dates only."],
  ["$like", "applies a `like` filter. Useful for strings only."],
  ["$re", "applies a regex filter. Useful for strings only."],
  ["$ilike", "applies a case-insensitive `like` filter. Useful for strings only."],
  ["$fulltext", "filters on full-text properties."],
  ["$overlap", "filters arrays that have values overlapping with the parameter."],
  ["$contains", "filters arrays that contain some of the values of the parameter."],
  ["$contained", "filters arrays that contain all the values of the parameter."],
  ["$exists", "filters by whether a value for the parameter exists (is not `null`)."],
]

const codeTabs = (tabs) =>
  `<CodeTabs group="api-intro">\n${tabs
    .map(([label, value, lang, code]) => `  <CodeTab label=${JSON.stringify(label)} value=${JSON.stringify(value)}>\n\n${fence(lang, code)}\n\n  </CodeTab>`)
    .join("\n")}\n</CodeTabs>`

function metadataSection(area) {
  return `## Manage Metadata

Many data models, such as products and carts, have a \`metadata\` field that stores custom information as key-value pairs.

When you set or update the \`metadata\` field through the API routes, the new metadata is merged with the existing metadata.

### Accepted Values in Metadata

The \`metadata\` is an object of key-value pairs. The keys are strings, and the values can be strings, numbers, booleans, dates, objects, or arrays of any of these. An empty string deletes the property from the metadata. The \`metadata\` is not validated, so you can store any custom data in it.

${fence(
  "json",
  `{
  "metadata": {
    "category": "electronics",
    "views": 1500,
    "is_featured": true,
    "tags": ["new", "sale"],
    "details": {
      "warranty": "2 years",
      "origin": "USA"
    }
  }
}`
)}

### Add or Update a Property in Metadata

To add or update a property, pass it in the request body as a key-value pair. Existing properties are not affected.

${codeTabs([
  [
    "JS SDK",
    "js-sdk",
    "ts",
    `sdk.admin.product.update("prod_123", {
  metadata: {
    new_property: "value",
  },
})`,
  ],
  [
    "cURL",
    "curl",
    "bash",
    `curl -X POST 'http://localhost:9000/admin/products/prod_123' \\
-H 'Authorization: Bearer {jwt_token}' \\
-H 'Content-Type: application/json' \\
--data-raw '{
  "metadata": { "new_property": "value" }
}'`,
  ],
])}

### Update Nested Objects in Metadata

Nested objects are not merged. When you update a nested object, pass the entire object, otherwise the properties you omit are removed.

### Remove a Property from Metadata

To remove a property, pass it with an empty string as its value. Other properties are not affected.

${fence(
  "json",
  `{
  "metadata": {
    "property_to_remove": ""
  }
}`
)}`
}

function fieldsSection(area) {
  const route = area === "admin" ? "/admin/products" : "/store/products"
  const sdk = area === "admin" ? "sdk.admin.product.list" : "sdk.store.product.list"
  const header = area === "store" ? ` \\\n-H 'x-publishable-api-key: {publishable_api_key}'` : ` \\\n-H 'Authorization: Bearer {jwt_token}'`
  const curl = (q) => `curl '${`http://localhost:9000${route}?${q}`}'${header}`
  return `## Select Fields and Relations

Many API routes accept a \`fields\` query parameter that selects which fields and relations are returned in a record. Fields and relations are separated by a comma.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `${sdk}({\n  fields: "title,handle",\n})`],
  ["cURL", "curl", "bash", curl("fields=title,handle")],
])}

This returns only the \`title\` and \`handle\` fields of each product, along with the fields that identify the record.

### Fields Operator

By default, only the selected fields and relations are returned. Before a field or relation, you can pass one of these operators:

- \`+\`: adds the field to the fields returned by default. For example, \`+title\` returns \`title\` along with the default fields.
- \`-\`: removes the field from the fields returned by default. For example, \`-handle\` removes \`handle\` from the default fields.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `${sdk}({\n  fields: "+title,-handle",\n})`],
  ["cURL", "curl", "bash", curl("fields=%2Btitle,-handle")],
])}

### Select Relations

To select a relation, pass its name prefixed by \`*\`. To select several relations, separate them with commas.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `${sdk}({\n  fields: "*variants,*options",\n})`],
  ["cURL", "curl", "bash", curl("fields=*variants,*options")],
])}

### Select Fields in a Relation

The \`*\` prefix selects all the fields of the relation. To select specific fields, use the \`<relation>.<field>\` format, such as \`variants.title\`.

### Select Nested Relations

A relation can have relations of its own. Chain the relation names with a dot, and prefix the deepest relation with \`*\` to select all its fields. For example, \`*variants.prices\`. Prefixing a relation with \`*\` only selects that relation's own fields, so a nested relation must be referenced explicitly through its parent.

### Select Custom Linked Data Models

Most API routes that accept \`fields\` allow you to select data models that a [module link](/learn/fundamentals/module-links) connects to the core data models. For example, if you linked a \`Brand\` to a \`Product\`, pass \`brand\` in \`fields\` to retrieve the brand of each product.

Some API routes restrict the fields and relations that you can retrieve. In that case, the application must allow the extra fields explicitly.`
}

function paramTypesSection(area) {
  const sdk = area === "admin" ? "sdk.admin.product.list" : "sdk.store.product.list"
  const base = `http://localhost:9000/${area}/products`
  const header = area === "store" ? ` \\\n-H 'x-publishable-api-key: {publishable_api_key}'` : ` \\\n-H 'Authorization: Bearer {jwt_token}'`
  const curl = (q, g = false) => `curl ${g ? "-g " : ""}"${base}?${q}"${header}`
  return `## Query Parameter Types

This section covers how to pass common data types as query parameters. It is useful if you send requests with cURL or tools like Postman. When you use the JS SDK, you can pass strings, numbers, booleans, arrays and objects directly, and the SDK encodes them.

### Strings

Pass a string as \`<parameter_name>=<value>\`. Encode characters other than letters and numbers. For example, encode a space as \`+\` or \`%20\`.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `${sdk}({\n  title: "Blue Shirt",\n})`],
  ["cURL", "curl", "bash", curl("title=Blue%20Shirt")],
])}

### Integers and Booleans

Pass them as \`<parameter_name>=<value>\`, such as \`offset=1\` or \`is_giftcard=true\`.

### Date and DateTime

Pass a date as \`<parameter_name>=<value>\` in the \`YYYY-MM-DD\` format, or a date and time in the \`YYYY-MM-DDTHH:MM:SSZ\` format. The \`T\` and \`Z\` are fixed.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `${sdk}({\n  created_at: { $lt: "2023-02-17T07:22:30Z" },\n})`],
  ["cURL", "curl", "bash", curl("created_at[$lt]=2023-02-17T07:22:30Z", true)],
])}

### Arrays

Pass an array in one of these forms:

- \`<parameter_name>[]=<value1>,<value2>\`, separating the values with a comma.
- \`<parameter_name>[]=<value1>&<parameter_name>[]=<value2>\`, passing each value as a separate query parameter. You can also set the index of each value in the brackets: \`<parameter_name>[0]=<value>\`.

The \`-g\` option of \`curl\` disables the errors that curl throws for the brackets.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `${sdk}({\n  sales_channel_id: ["sc_123", "sc_456"],\n})`],
  ["cURL", "curl", "bash", curl("sales_channel_id[]=sc_123,sc_456", true)],
])}

### Objects

Pass an object as separate query parameters in the form \`<parameter_name>[<key>]=<value>\`.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `${sdk}({\n  created_at: { $lt: "2023-02-17", $gt: "2022-09-17" },\n})`],
  ["cURL", "curl", "bash", curl("created_at[$lt]=2023-02-17&created_at[$gt]=2022-09-17", true)],
])}

### Filter Operators

Filter parameters, such as \`created_at\` or \`email\`, accept either a plain value or an object whose keys are filter operators. The operators are available on the filter parameters of the list API routes:

${FILTER_OPERATORS.map(([op, text]) => `- \`${op}\`: ${text}`).join("\n")}

For example, \`created_at[$gt]=2022-09-17&created_at[$lt]=2023-02-17\` returns the records created between the two dates.`
}

function paginationSection(area) {
  const sdk = area === "admin" ? "sdk.admin.product.list" : "sdk.store.product.list"
  const route = `http://localhost:9000/${area}/products`
  const header = area === "store" ? ` \\\n-H 'x-publishable-api-key: {publishable_api_key}'` : ` \\\n-H 'Authorization: Bearer {jwt_token}'`
  return `## Pagination

### Query Parameters

List API routes, such as the ones that list customers or products, accept the \`limit\` and \`offset\` query parameters.

- \`limit\` is the maximum number of items to return.
- \`offset\` is the number of items to skip before returning the records.

Use \`offset\` to change between pages. For example, if the limit is \`50\`, the offset of page \`1\` is \`0\` and the offset of page \`2\` is \`50\`.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `${sdk}({\n  limit: 5,\n  offset: 0,\n})`],
  ["cURL", "curl", "bash", `curl "${route}?limit=5&offset=0"${header}`],
])}

### Response Fields

The response of a list API route has these pagination fields next to the records:

- \`limit\`: the maximum number of items that can be returned.
- \`offset\`: the number of items that were skipped.
- \`count\`: the total number of items available. Divide it by the \`limit\` to get the number of pages.

### Sort Order

The \`order\` query parameter, available on API routes that support pagination, sorts the items by one of their fields. The default direction is ascending. To sort in descending order, prefix the field name with a dash (\`-\`).

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `${sdk}({\n  order: "-created_at",\n})`],
  ["cURL", "curl", "bash", `curl "${route}?order=-created_at"${header}`],
])}`
}

function errorsSection() {
  return `## Errors

When a request fails, the API route responds with a status code in the \`4xx\` or \`5xx\` range and a JSON body that describes the error:

${fence(
  "json",
  `{
  "type": "not_found",
  "message": "Product with id: prod_123 was not found"
}`
)}

The \`type\` field identifies the kind of error, and \`message\` describes it. Some errors also have a \`code\` field. The status code follows from the error type:

<Table>
  <Table.Header>
    <Table.Row>
      <Table.HeaderCell>Status</Table.HeaderCell>
      <Table.HeaderCell>Error types</Table.HeaderCell>
    </Table.Row>
  </Table.Header>
  <Table.Body>
    <Table.Row>
      <Table.Cell>\`400\`</Table.Cell>
      <Table.Cell>\`invalid_data\`, \`not_allowed\`</Table.Cell>
    </Table.Row>
    <Table.Row>
      <Table.Cell>\`401\`</Table.Cell>
      <Table.Cell>\`unauthorized\`</Table.Cell>
    </Table.Row>
    <Table.Row>
      <Table.Cell>\`403\`</Table.Cell>
      <Table.Cell>\`forbidden\`</Table.Cell>
    </Table.Row>
    <Table.Row>
      <Table.Cell>\`404\`</Table.Cell>
      <Table.Cell>\`not_found\`</Table.Cell>
    </Table.Row>
    <Table.Row>
      <Table.Cell>\`409\`</Table.Cell>
      <Table.Cell>\`conflict\`</Table.Cell>
    </Table.Row>
    <Table.Row>
      <Table.Cell>\`422\`</Table.Cell>
      <Table.Cell>\`duplicate_error\`, \`payment_authorization_error\`</Table.Cell>
    </Table.Row>
    <Table.Row>
      <Table.Cell>\`500\`</Table.Cell>
      <Table.Cell>\`database_error\`, \`unexpected_state\`, \`invalid_argument\`, \`unknown_error\`</Table.Cell>
    </Table.Row>
  </Table.Body>
</Table>`
}

function authSection(area) {
  const jwtRoute = area === "admin" ? "/auth/user/emailpass" : "/auth/customer/emailpass"
  const actor = area === "admin" ? "user" : "customer"
  const email = area === "admin" ? "user@example.com" : "customer@example.com"
  const sessionLink = siteLink("JS SDK configuration", "/resources/js-sdk")
  const parts = []
  if (area === "admin") {
    parts.push(`## Authentication

There are three ways to send authenticated requests to the MedusaGo server: with a JWT token in a bearer authorization header, with a secret API key, or with a cookie session ID.`)
  } else {
    parts.push(`## Authentication

Some Store API routes, such as the ones that manage the logged-in customer, require the customer to be authenticated. There are two ways to send authenticated requests: with a JWT token in a bearer authorization header, or with a cookie session ID. Routes that read public data, such as the products and the cart routes, do not require authentication.`)
  }
  parts.push(`### 1. Bearer Authorization with JWT Tokens

Use a JWT token in the bearer authorization header of a request. The client manages the authentication state, which suits single-page applications, Jamstack storefronts and mobile applications.

#### How to Obtain the JWT Token

Send a request to the authentication route, passing the ${actor}'s email and password in the request body.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `const token = await sdk.auth.login("${actor}", "emailpass", {\n  email,\n  password,\n})`],
  [
    "cURL",
    "curl",
    "bash",
    `curl -X POST 'http://localhost:9000${jwtRoute}' \\
-H 'Content-Type: application/json' \\
--data-raw '{
  "email": "${email}",
  "password": "supersecret"
}'`,
  ],
])}

If the authentication succeeds, the response has a \`token\` property that holds the JWT token.

${fence("json", `{\n  "token": "123..."\n}`)}

#### How to Use the JWT Token

Pass the JWT token in the \`Authorization: Bearer\` header of the requests.

${fence("bash", "Authorization: Bearer {jwt_token}")}

If you use the JS SDK, the \`login\` method sets the token and passes it to subsequent requests, provided that the \`auth.type\` option of the SDK is \`jwt\`. Learn more in the ${sessionLink} documentation.`)
  if (area === "admin") {
    parts.push(`### 2. Secret API Key

Use a secret API key to send authenticated requests as an admin user.

#### How to Create a Secret API Key

Create the key in the Medusa Admin (${siteLink("Secret API Keys", "/user-guide/settings/developer/secret-api-keys")}) or with the [Create API Key](/api/admin/api-keys) API route. An \`api_key\` object is returned in the response, and its \`token\` property is the secret key.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `const { api_key } = await sdk.admin.apiKey.create({\n  title: "My Token",\n  type: "secret",\n})\n\nconsole.log(api_key.token)`],
  [
    "cURL",
    "curl",
    "bash",
    `curl -X POST 'http://localhost:9000/admin/api-keys' \\
-H 'Content-Type: application/json' \\
-H 'Authorization: Bearer {jwt_token}' \\
--data '{
  "title": "my token",
  "type": "secret"
}'`,
  ],
])}

#### How to Use the Secret API Key

Pass the key in the \`Authorization\` header using the Basic scheme.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `import Medusa from "@medusajs/js-sdk"\n\nexport const sdk = new Medusa({\n  // other options...\n  apiKey: "{api_key_token}",\n})`],
  ["cURL", "curl", "bash", `curl 'http://localhost:9000/admin/products' \\\n-H 'Authorization: Basic {api_key_token}'`],
])}

A secret API key sent as a bearer token is rejected with a \`401\` status code.

### 3. Cookie Session ID`)
  } else {
    parts.push(`### 2. Cookie Session ID`)
  }
  parts.push(`When you authenticate and create a cookie session, the browser passes the session cookie (\`connect.sid\`) with every request, as do tools like Postman.

#### How to Obtain the Cookie Session

You first need a [JWT token](#1-bearer-authorization-with-jwt-tokens). Then, with the JS SDK, set the \`auth.type\` option to \`session\`: the \`auth.login\` method creates the session cookie and passes it in the following requests.

Without the JS SDK, send a request to the \`/auth/session\` route with the JWT token. Pass the \`-v\` option to \`curl\` to view the response headers.

${fence("bash", `curl -v -X POST 'http://localhost:9000/auth/session' \\\n--header 'Authorization: Bearer {jwt_token}'`)}

The \`Set-Cookie\` header of the response has the cookie session ID.

${fence("bash", "Set-Cookie: connect.sid=s%3A2Bu8B...;")}

#### How to Use the Cookie Session ID

Copy the value after \`connect.sid=\` (without the trailing \`;\`) and pass it in the \`Cookie\` header of the following requests.

${fence("bash", `curl 'http://localhost:9000/${area}/${area === "admin" ? "products" : "customers/me"}' \\\n-H 'Cookie: connect.sid={sid}'`)}

When you send requests with the Fetch API from a browser, pass the \`credentials\` option with the value \`include\`.

${fence("ts", `fetch("http://localhost:9000/${area}/${area === "admin" ? "products" : "customers/me"}", {\n  credentials: "include",\n})`)}`)
  return parts.join("\n\n")
}

function publishableKeySection() {
  const guide = siteLink("Publishable API Keys", "/resources/storefront-development/publishable-api-keys")
  const admin = siteLink("Medusa Admin", "/user-guide/settings/developer/publishable-api-keys")
  return `## Publishable API Key

The Store API routes require a publishable API key in the \`x-publishable-api-key\` header. A publishable key scopes the requests to one or more sales channels, so the storefront only sees the products, carts and stock of those channels. A request without a valid key is rejected with a \`400\` status code.

### How to Create a Publishable API Key

Create the key in the ${admin}, or with the [Create API Key](/api/admin/api-keys) route of the Admin API using the \`publishable\` type. Learn more in the ${guide} guide.

### How to Use a Publishable API Key

Pass the key in the \`x-publishable-api-key\` header.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `import Medusa from "@medusajs/js-sdk"\n\nexport const sdk = new Medusa({\n  // other options...\n  publishableKey: "{publishable_api_key}",\n})`],
  ["cURL", "curl", "bash", `curl 'http://localhost:9000/store/products' \\\n-H 'x-publishable-api-key: {publishable_api_key}'`],
])}`
}

function localizationSection() {
  const mod = siteLink("Translation Module", "/resources/commerce-modules/translation")
  return `## Localization

<Note>

Localization requires the \`translation\` feature flag. Set the \`MEDUSA_FF_TRANSLATION\` environment variable to \`true\` to enable it. See the ${mod} documentation.

</Note>

The Store API routes can return translated data. By default, they return the original data without localization.

You can set the locale of a request in one of these ways:

- With the JS SDK, call its \`setLocale\` method. The SDK then sends the locale in the \`x-medusa-locale\` header of the following requests.
- Pass the \`locale\` query parameter to a \`/store\` API route.
- Set the \`x-medusa-locale\` header of the request.

Locales use the IETF BCP 47 format, such as \`en-US\` or \`fr-FR\`. The data is returned in the requested locale when a translation exists, and falls back to the original data otherwise.

${codeTabs([
  ["JS SDK", "js-sdk", "ts", `sdk.setLocale("fr-FR")\n\nconst { products } = await sdk.store.product.list()`],
  ["Query Parameter", "query-param", "bash", `curl "http://localhost:9000/store/products?locale=fr-FR" \\\n-H 'x-publishable-api-key: {publishable_api_key}'`],
  ["Header", "header", "bash", `curl "http://localhost:9000/store/products" \\\n-H 'x-publishable-api-key: {publishable_api_key}' \\\n-H 'x-medusa-locale: fr-FR'`],
])}`
}

function renderIntro(area, tagPages) {
  const isAdmin = area.key === "admin"
  const title = isAdmin ? "Admin API Reference" : "Store API Reference"
  const intro = isAdmin
    ? `This reference documents the Admin REST API of the MedusaGo server. The Admin API performs administrative functionality, and it is what the Medusa Admin and any custom admin dashboard use to read and change the data of your store.

All API routes of this reference are prefixed with \`/admin\`, except for the authentication routes, which are prefixed with \`/auth\`. During development, they are available under \`http://localhost:9000\`. In production, replace it with the URL of your MedusaGo server.

The HTTP API of MedusaGo is the same as the one of Medusa v2, so you can use the [Medusa JS SDK](${siteHas("/resources/js-sdk") ? "/resources/js-sdk" : "https://docs.medusajs.com/resources/js-sdk"}) and any tool that targets the Medusa Admin API as they are.`
    : `This reference documents the Store REST API of the MedusaGo server. The Store API powers storefronts: it lets customers browse products, manage carts, check out, and manage their account and orders.

All API routes of this reference are prefixed with \`/store\`, except for the authentication routes, which are prefixed with \`/auth\`. During development, they are available under \`http://localhost:9000\`. In production, replace it with the URL of your MedusaGo server.

The HTTP API of MedusaGo is the same as the one of Medusa v2, so you can use the [Medusa JS SDK](${siteHas("/resources/js-sdk") ? "/resources/js-sdk" : "https://docs.medusajs.com/resources/js-sdk"}) and any storefront that targets the Medusa Store API as they are.`

  const sections = [
    `---\ngenerate_toc: true\n---\n\nimport { CodeTabs, CodeTab, Table } from "docs-ui"\n\nexport const metadata = {\n  title: \`${title}\`,\n}\n\n# {metadata.title}\n\n${intro}`,
    authSection(area.key),
  ]
  if (!isAdmin) sections.push(publishableKeySection())
  sections.push(metadataSection(area.key), fieldsSection(area.key), paramTypesSection(area.key), paginationSection(area.key))
  if (!isAdmin) sections.push(localizationSection())
  sections.push(errorsSection())
  sections.push(
    `## ${isAdmin ? "Admin" : "Store"} API Routes\n\nThe API routes are grouped by the resource they manage:\n\n${tagPages
      .map((t) => `- [${escapeMdx(t.name)}](/api/${area.key}/${t.slug})`)
      .join("\n")}`
  )
  return sections.join("\n\n") + "\n"
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

const stats = { areas: {}, droppedEnterprise: [], droppedNotRegistered: [] }
const sidebars = []
const writes = new Map() // file -> content

for (const area of AREAS) {
  const spec = YAML.parse(fs.readFileSync(path.join(specsDir, area.key, "openapi.full.yaml"), "utf8"))
  const tools = makeSchemaTools(spec, area.key)
  const ctx = { area: area.key, tools, spec }

  // operations by tag, in the spec's order
  const byTag = new Map()
  let total = 0
  for (const [specPath, item] of Object.entries(spec.paths)) {
    for (const [m, op] of Object.entries(item)) {
      if (!op || typeof op !== "object" || !op.operationId) continue
      const method = m.toUpperCase()
      const tag = (op.tags || [])[0]
      total++
      if (isEnterprise(specPath, op) || (op.tags || []).some((t) => /^\[.*\]$/.test(t))) {
        stats.droppedEnterprise.push(`${area.key} ${method} ${specPath}`)
        continue
      }
      const route = findGoRoute(method, specPath)
      if (!route) {
        stats.droppedNotRegistered.push(`${area.key} ${method} ${specPath}`)
        continue
      }
      if (!byTag.has(tag)) byTag.set(tag, [])
      byTag.get(tag).push({ method, specPath, op, goFlag: route.flag })
    }
  }

  const tagInfo = new Map((spec.tags || []).map((t) => [t.name, t]))
  const tagPages = []
  let opCount = 0
  for (const [tagName, entries] of byTag) {
    const info = tagInfo.get(tagName) || { name: tagName }
    const slug = slugify(tagName)
    tagPages.push({ name: tagName, slug, entries, info })
    opCount += entries.length
  }
  tagPages.sort((a, b) => a.name.localeCompare(b.name, "en"))

  for (const page of tagPages) {
    const title = `${page.name} API Routes`
    const description = page.info.description ? mdxText(page.info.description, area.key) : ""
    const body = page.entries.map((e) => renderOperation(ctx, e)).join("\n\n---\n\n")
    const usesTypes = body.includes("<TypeList")
    const usesTabs = body.includes("<CodeTabs")
    const usesTable = false
    const imports = [usesTabs ? "CodeTabs, CodeTab" : "", usesTypes ? "TypeList" : "", usesTable ? "Table" : ""].filter(Boolean).join(", ")
    const content = `---\ngenerate_toc: true\n---\n\n${imports ? `import { ${imports} } from "docs-ui"\n\n` : ""}export const metadata = {\n  title: \`${title}\`,\n}\n\n# {metadata.title}\n\n${description ? description + "\n\n" : ""}${body}\n`
    writes.set(path.join(outDir, area.key, page.slug, "page.mdx"), content)
  }

  writes.set(path.join(outDir, area.key, "page.mdx"), renderIntro(area, tagPages))

  sidebars.push({
    sidebar_id: area.key,
    title: area.sidebarTitle,
    items: [
      { loaded: true, type: "link", path: `/api/${area.key}`, title: "Introduction", children: [] },
      { type: "separator" },
      ...tagPages.map((t) => ({ loaded: true, type: "link", path: `/api/${area.key}/${t.slug}`, title: t.name, description: "", children: [] })),
    ],
  })
  stats.areas[area.key] = { specOperations: total, tags: tagPages.length, operations: opCount }
}

writes.set(
  path.join(outDir, "page.mdx"),
  `---
generate_toc: true
---

export const metadata = {
  title: \`MedusaGo API Reference\`,
}

# {metadata.title}

MedusaGo exposes two REST APIs. Both are identical to the ones of Medusa v2, so the Medusa JS SDK, the Medusa Admin and storefronts built for Medusa work with MedusaGo without changes.

- [Admin API](/api/admin): manage the data of your store. It is used by the Medusa Admin, and by your own admin tools and integrations.
- [Store API](/api/store): build storefronts. It lets customers browse products, manage carts, check out and manage their account.

The reference lists the routes that the MedusaGo server serves. For each API route, you find its parameters, request body, response, and request examples with cURL and the JS SDK.
`
)

// Replace the previous output so removed tags do not leave stale pages.
fs.rmSync(outDir, { recursive: true, force: true })
for (const [file, content] of writes) {
  fs.mkdirSync(path.dirname(file), { recursive: true })
  fs.writeFileSync(file, content)
}
fs.writeFileSync(path.join(root, "src/generated/api-sidebar.mjs"), `export const generatedSidebars = ${JSON.stringify(sidebars, null, 2)}\n`)

console.log("generated api reference:")
for (const [k, v] of Object.entries(stats.areas)) console.log(`  ${k}: ${v.tags} tags, ${v.operations} operations (${v.specOperations} in the spec)`)
console.log(`  dropped (enterprise): ${stats.droppedEnterprise.length}`)
for (const l of stats.droppedEnterprise) console.log(`    ${l}`)
console.log(`  dropped (not registered in Go): ${stats.droppedNotRegistered.length}`)
for (const l of stats.droppedNotRegistered) console.log(`    ${l}`)
