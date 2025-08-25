// internal/adapters/discord/policy.go
// Minimal privilege check based on ADMIN_ROLE_IDS env or Administrator permission.

package discord

import (
	"os"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

var (
	adminOnce   sync.Once
	adminRoleID = map[string]struct{}{}
)

func loadAdminRolesFromEnv() {
	raw := os.Getenv("ADMIN_ROLE_IDS")
	for _, id := range strings.Split(raw, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			adminRoleID[id] = struct{}{}
		}
	}
}

// IsPrivileged returns true if the member has Administrator or one of ADMIN_ROLE_IDS.
func IsPrivileged(i *discordgo.InteractionCreate) bool {
	adminOnce.Do(loadAdminRolesFromEnv)
	if i.Member == nil {
		return false
	}
	if i.Member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}
	for _, r := range i.Member.Roles {
		if _, ok := adminRoleID[r]; ok {
			return true
		}
	}
	return false
}

// RequirePrivileged replies ephemeral and returns false if not privileged.
func RequirePrivileged(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if IsPrivileged(i) {
		return true
	}
	_ = SendEphemeral(s, i, "⛔ You don't have permission for this action.")
	return false
}
