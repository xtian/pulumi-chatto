package provider

import (
	"reflect"
	"testing"
)

func TestBotPermissionIDRoundTrip(t *testing.T) {
	group, server := ScopeKindGroup, ScopeKindServer
	groupID := "Gabc"

	for _, args := range []BotPermissionArgs{
		{BotUserID: "Ubot", Permission: "role.assign", ScopeKind: &server},
		{BotUserID: "Ubot", Permission: "room.join", ScopeID: &groupID, ScopeKind: &group},
	} {
		id := botPermissionID(args)
		parsed, err := parseBotPermissionID(id)
		if err != nil {
			t.Fatalf("parseBotPermissionID(%q) error = %v", id, err)
		}
		if !reflect.DeepEqual(parsed, args) {
			t.Fatalf("parseBotPermissionID(%q) = %+v, want %+v", id, parsed, args)
		}
	}
}

func TestCheckScopeRejectsMismatchedScopeIDs(t *testing.T) {
	group, server := ScopeKindGroup, ScopeKindServer
	groupID := "Gabc"

	if failures := checkScope("room.join", &group, nil, false); len(failures) != 1 {
		t.Fatalf("checkScope(group without ID) = %v, want one failure", failures)
	}
	if failures := checkScope("room.join", &server, &groupID, false); len(failures) != 1 {
		t.Fatalf("checkScope(server with ID) = %v, want one failure", failures)
	}
	if failures := checkScope("room.join", &group, &groupID, false); len(failures) != 0 {
		t.Fatalf("checkScope(group with ID) = %v, want none", failures)
	}
	if failures := checkScope("room.join", &group, nil, true); len(failures) != 0 {
		t.Fatalf("checkScope(group with computed ID) = %v, want none", failures)
	}
}
