package provider

import (
	"reflect"
	"testing"
)

func TestRolePermissionIDRoundTrip(t *testing.T) {
	group, room, server := ScopeKindGroup, ScopeKindRoom, ScopeKindServer
	groupID, roomID := "Gabc", "Rxyz"

	tests := []struct {
		name string
		args RolePermissionArgs
		id   string
	}{
		{
			name: "server scope",
			args: RolePermissionArgs{Permission: "room.manage", RoleName: "tech", ScopeKind: &server},
			id:   "tech/room.manage/server",
		},
		{
			name: "group scope",
			args: RolePermissionArgs{Permission: "room.create", RoleName: "tech", ScopeID: &groupID, ScopeKind: &group},
			id:   "tech/room.create/group/Gabc",
		},
		{
			name: "room scope",
			args: RolePermissionArgs{Permission: "message.pin", RoleName: "everyone", ScopeID: &roomID, ScopeKind: &room},
			id:   "everyone/message.pin/room/Rxyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rolePermissionID(tt.args); got != tt.id {
				t.Fatalf("rolePermissionID() = %q, want %q", got, tt.id)
			}

			parsed, err := parseRolePermissionID(tt.id)
			if err != nil {
				t.Fatalf("parseRolePermissionID(%q) error = %v", tt.id, err)
			}
			if !reflect.DeepEqual(parsed, tt.args) {
				t.Fatalf("parseRolePermissionID(%q) = %+v, want %+v", tt.id, parsed, tt.args)
			}
		})
	}
}

func TestRolePermissionIDDefaultsToServerScope(t *testing.T) {
	args := RolePermissionArgs{Permission: "room.manage", RoleName: "tech"}
	if got, want := rolePermissionID(args), "tech/room.manage/server"; got != want {
		t.Fatalf("rolePermissionID() = %q, want %q", got, want)
	}
}

func TestParseRolePermissionIDRejectsMalformedIDs(t *testing.T) {
	for _, id := range []string{
		"",
		"tech",
		"tech/room.manage",
		"tech/room.manage/galaxy",
		"tech/room.manage/room/Rxyz/extra",
	} {
		t.Run(id, func(t *testing.T) {
			if _, err := parseRolePermissionID(id); err == nil {
				t.Fatalf("parseRolePermissionID(%q) error = nil, want an error", id)
			}
		})
	}
}
