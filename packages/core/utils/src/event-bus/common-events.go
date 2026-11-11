package eventbus

// CommonEvents mirrors the CommonEvents enum.
var CommonEvents = struct {
	CREATED, UPDATED, DELETED, RESTORED, ATTACHED, DETACHED string
}{
	CREATED:  "created",
	UPDATED:  "updated",
	DELETED:  "deleted",
	RESTORED: "restored",
	ATTACHED: "attached",
	DETACHED: "detached",
}

// commonEventsOrder is Object.values(CommonEvents).
var commonEventsOrder = []string{"created", "updated", "deleted", "restored", "attached", "detached"}
