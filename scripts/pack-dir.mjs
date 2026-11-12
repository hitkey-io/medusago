// Packs a directory into a deterministic .tar.gz (ustar, files sorted, no
// timestamps or owners): node scripts/pack-dir.mjs <dir> <out.tar.gz>.
import fs from "node:fs"
import path from "node:path"
import zlib from "node:zlib"

const [dir, out] = process.argv.slice(2)

function walk(root, rel = "") {
  return fs
    .readdirSync(path.join(root, rel), { withFileTypes: true })
    .flatMap((d) => {
      const p = rel ? `${rel}/${d.name}` : d.name
      return d.isDirectory() ? walk(root, p) : [p]
    })
}

function header(name, size) {
  if (Buffer.byteLength(name) > 100) {
    throw new Error(`path too long for ustar: ${name}`)
  }
  const h = Buffer.alloc(512)
  h.write(name, 0, 100)
  h.write("0000644\0", 100)
  h.write("0000000\0", 108)
  h.write("0000000\0", 116)
  h.write(size.toString(8).padStart(11, "0") + "\0", 124)
  h.write("00000000000\0", 136)
  h.write("        ", 148)
  h.write("0", 156)
  h.write("ustar\0", 257)
  h.write("00", 263)
  let sum = 0
  for (const b of h) sum += b
  h.write(sum.toString(8).padStart(6, "0") + "\0 ", 148)
  return h
}

const parts = []
for (const rel of walk(dir).sort()) {
  const data = fs.readFileSync(path.join(dir, rel))
  parts.push(header(rel, data.length), data)
  const pad = (512 - (data.length % 512)) % 512
  if (pad) parts.push(Buffer.alloc(pad))
}
parts.push(Buffer.alloc(1024))
fs.writeFileSync(out, zlib.gzipSync(Buffer.concat(parts), { level: 9 }))
