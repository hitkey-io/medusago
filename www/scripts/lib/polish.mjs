// Shared by import-upstream.mjs and polish-text.mjs.
// MedusaGo terminology. Applied to prose only (never inside code fences). "Medusa"
// stays where it names the upstream project, its API, the Admin or the UI kit.
export const TERMS = [
  [/`medusa-config\.(ts|js)`/g, "`medusa-config.go`"],
  [/(?<![/\w-])create-medusa-app\b(?!-errors)/g, "create-medusago-app"],
  [/\bMedusa (application|applications|server|backend|project|projects|instance|instances|installation)\b/gi, "MedusaGo $1"],
  [/\bMedusa CLI\b/g, "MedusaGo CLI"],
  [/\bMedusa (store|stores|workflows|plugins)\b/g, "MedusaGo $1"],
  // the product as the subject of a sentence ("Medusa will calculate taxes", "if Medusa manages the inventory")
  [/\bMedusa (?=(?:will|automatically|allows|supports|manages|keeps|adds|calculates|comes|runs|couldn[’']t|hasn[’']t|doesn[’']t|then|is currently|creates|reserves|allocates|to track)\b)/g, "MedusaGo "],
  [/\b(outside of|into|in|exploring|customize) Medusa\b(?! (Admin|v\d))/g, "$1 MedusaGo"],
  // an upstream typo: the section is called Media
  [/under the Medusa section/g, "under the Media section"],
]
export function applyTerms(text) {
  let inFence = false
  return text
    .split("\n")
    .map((line) => {
      if (/^\s*```/.test(line)) inFence = !inFence
      if (inFence || /^\s*```/.test(line)) return line
      return TERMS.reduce((l, [re, to]) => l.replace(re, to), line)
    })
    .join("\n")
}

// Notes that only say in which Medusa version a feature appeared ("available from Medusa v2.12.3, ask your technical team to upgrade") do not apply to
// MedusaGo, which ships the whole feature set of its baseline version. Notes that carry other information are kept.
const VERSION_NOTE = /releases\/tag\/v\d|earlier than v2\.\d|update your Medusa version|Prior to \[Medusa/
// A note is dropped only when it is just that statement: short, and not a titled piece of advice ("Recommended Approach").
const KEEP_TITLE = /<Note[^>]*title="(?!Prerequisite)[^"]*"/
export function dropVersionNotes(text) {
  return text
    .replace(/\n*<Note[^>]*>\n[\s\S]*?\n<\/Note>\n/g, (block) => {
      if (!VERSION_NOTE.test(block)) return block
      if (KEEP_TITLE.test(block) || block.length > 650) return block
      return "\n"
    })
    .replace(/\n{3,}/g, "\n\n")
}

// A line that is only a sentence about the release a feature appeared in ("This component is available since Medusa [v2.21.0](...).").
const VERSION_LINE = /^\s*(?:- )?[A-Z][^\[\n]{0,110}\b(?:available|introduced|added)(?: in| since| from| starting from| starting| after)? (?:Medusa )?\[[^\]]*\]\([^)]*releases\/tag[^)]*\)\.?\s*$/
export function dropVersionLines(text) {
  let inFence = false
  return text
    .split("\n")
    .filter((line) => {
      if (/^\s*```/.test(line)) inFence = !inFence
      return inFence || !VERSION_LINE.test(line)
    })
    .join("\n")
    .replace(/\n{3,}/g, "\n\n")
}
