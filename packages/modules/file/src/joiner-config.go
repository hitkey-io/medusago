package file

import "github.com/hitkey-io/medusago/packages/core/types/src/joiner"

// JoinerConfig mirrors joinerConfig (defineJoinerConfig(Modules.FILE,
// { models: [{ name: "File" }] })).
var JoinerConfig = joiner.MustParseModuleJoinerConfig(`{
  "serviceName": "file",
  "primaryKeys": ["id"],
  "idPrefixToEntityName": {},
  "linkableKeys": { "file_id": "File" },
  "alias": [
    { "name": ["file", "files"], "entity": "File", "args": { "methodSuffix": "Files" } }
  ]
}`)
