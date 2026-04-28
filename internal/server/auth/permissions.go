package auth

import "github.com/lingyuins/octopus/internal/model"

// Permission represents a fine-grained access right.
type Permission string

const (
	PermChannelsRead  Permission = "channels:read"
	PermChannelsWrite Permission = "channels:write"
	PermGroupsRead    Permission = "groups:read"
	PermGroupsWrite   Permission = "groups:write"
	PermAPIKeysRead   Permission = "apikeys:read"
	PermAPIKeysWrite  Permission = "apikeys:write"
	PermSettingsRead  Permission = "settings:read"
	PermSettingsWrite Permission = "settings:write"
	PermLogsRead      Permission = "logs:read"
	PermLogsWrite     Permission = "logs:write"
	PermStatsRead     Permission = "stats:read"
	PermUsersRead     Permission = "users:read"
	PermUsersWrite    Permission = "users:write"
)

var adminPermissions = []Permission{
	PermChannelsRead, PermChannelsWrite,
	PermGroupsRead, PermGroupsWrite,
	PermAPIKeysRead, PermAPIKeysWrite,
	PermSettingsRead, PermSettingsWrite,
	PermLogsRead, PermLogsWrite, PermStatsRead,
	PermUsersRead, PermUsersWrite,
}

var editorPermissions = []Permission{
	PermChannelsRead, PermChannelsWrite,
	PermGroupsRead, PermGroupsWrite,
	PermAPIKeysRead, PermAPIKeysWrite,
	PermSettingsRead, PermSettingsWrite,
	PermLogsRead, PermLogsWrite, PermStatsRead,
}

var viewerPermissions = []Permission{
	PermChannelsRead,
	PermGroupsRead,
	PermAPIKeysRead,
	PermSettingsRead,
	PermLogsRead, PermStatsRead,
}

var rolePermissions = map[string][]Permission{
	model.UserRoleAdmin:  adminPermissions,
	model.UserRoleEditor: editorPermissions,
	model.UserRoleViewer: viewerPermissions,
}

// HasPermission checks if the given role has the specified permission.
func HasPermission(role string, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}
