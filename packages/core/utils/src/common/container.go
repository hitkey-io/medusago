package common

// ContainerRegistrationKeys mirrors ContainerRegistrationKeys.
var ContainerRegistrationKeys = struct {
	PG_CONNECTION       string
	MANAGER             string
	CONFIG_MODULE       string
	LOGGER              string
	REMOTE_QUERY        string
	QUERY               string
	REMOTE_LINK         string
	LINK                string
	FEATURE_FLAG_ROUTER string
}{
	PG_CONNECTION:       "__pg_connection__",
	MANAGER:             "manager",
	CONFIG_MODULE:       "configModule",
	LOGGER:              "logger",
	REMOTE_QUERY:        "remoteQuery",
	QUERY:               "query",
	REMOTE_LINK:         "remoteLink",
	LINK:                "link",
	FEATURE_FLAG_ROUTER: "featureFlagRouter",
}
