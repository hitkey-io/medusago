// parseMigration returns the statements of a migration generated upstream
// (TSMigrationGenerator output): the addSql arguments of up() and down(),
// with "" for the blank lines separating groups of statements. down is null
// when the migration has no down method.
export function parseMigration(code) {
  const section = (name) => {
    const header = `  override async ${name}(): Promise<void> {\n`
    const start = code.indexOf(header)
    if (start < 0) {
      return null
    }
    const out = []
    for (const line of code.slice(start + header.length).split("\n")) {
      if (line === "  }") {
        return out
      }
      if (line === "") {
        out.push("")
        continue
      }
      const m = line.match(/^ {4}this\.addSql\(`([\s\S]*)`\);$/)
      if (!m) {
        throw new Error(`unexpected migration line: ${line}`)
      }
      out.push(m[1].replace(/\\([`$\\])/g, "$1"))
    }
    throw new Error(`unterminated ${name} method`)
  }
  return { up: section("up"), down: section("down") }
}
