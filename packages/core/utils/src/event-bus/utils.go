package eventbus

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// BuildModuleResourceEventName mirrors buildModuleResourceEventName.
func BuildModuleResourceEventName(prefix, objectName, action string) string {
	return dal.BuildModuleResourceEventName(prefix, objectName, action)
}

// BuildEventNamesFromEntityName mirrors buildEventNamesFromEntityName:
// { SERVICE_ZONE_CREATED: "<prefix>.service-zone.created", ... } for every
// name and common event.
func BuildEventNamesFromEntityName(names []string, prefix string) map[string]string {
	events := map[string]string{}
	for _, name := range names {
		snake := strings.ToUpper(common.CamelToSnakeCase(name))
		for _, event := range commonEventsOrder {
			events[snake+"_"+strings.ToUpper(event)] = BuildModuleResourceEventName(prefix, name, event)
		}
	}
	return events
}
