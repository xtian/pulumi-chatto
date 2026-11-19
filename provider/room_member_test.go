package provider

import "testing"

func TestRoomMemberIDRoundTrip(t *testing.T) {
	args := RoomMemberArgs{RoomID: "R7ooYlq4dv59W6j", UserID: "U5kq2b7xS1fYh0a"}
	id := roomMemberID(args)

	parsed, err := parseRoomMemberID(id)
	if err != nil {
		t.Fatalf("parseRoomMemberID(%q) error = %v", id, err)
	}
	if parsed != args {
		t.Fatalf("parseRoomMemberID(%q) = %+v, want %+v", id, parsed, args)
	}

	for _, invalid := range []string{"Rabc", "Rabc/", "/Uabc", "Rabc/Uabc/extra"} {
		if _, err := parseRoomMemberID(invalid); err == nil {
			t.Fatalf("parseRoomMemberID(%q) succeeded, want error", invalid)
		}
	}
}
