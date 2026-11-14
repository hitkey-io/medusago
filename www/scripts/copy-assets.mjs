// Next.js serves each app's public/ under its basePath; here the shared images
// in assets/images are copied to public/<section>/images for every section.
import fs from "node:fs"
import path from "node:path"

const root = path.resolve(import.meta.dirname, "..")
const sections = fs.readdirSync(path.join(root, "content"))
for (const section of sections) {
  const dest = path.join(root, "public", section, "images")
  fs.mkdirSync(dest, { recursive: true })
  fs.cpSync(path.join(root, "assets/images"), dest, { recursive: true })
}
console.log("assets copied for", sections.join(", "))
