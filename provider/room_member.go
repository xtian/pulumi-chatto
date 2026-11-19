package provider

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/pulumi/pulumi-go-provider/infer"

	apiv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/api/v1"
)

// RoomMember adds a user or bot to a channel room as an explicit member.
// The resource ID is <roomId>/<userId>.
type RoomMember struct{}

// RoomMemberArgs are the inputs of a RoomMember.
type RoomMemberArgs struct {
	RoomID string `pulumi:"roomId" provider:"replaceOnChanges"`
	UserID string `pulumi:"userId" provider:"replaceOnChanges"`
}

// RoomMemberState is the stored state of a RoomMember.
type RoomMemberState struct {
	RoomMemberArgs
}

// Annotate describes the resource in the schema.
func (r *RoomMember) Annotate(a infer.Annotator) {
	a.Describe(&r, "Adds a user or bot to a channel room as an explicit member, for example so a bot can "+
		"post to a room that isn't universal. Adding someone else needs room.manage for the room or "+
		"user.manage-accounts. Universal, DM, and archived rooms can't take explicit members. Deleting the "+
		"resource removes the member. Import with `<roomId>/<userId>`.")
}

// Annotate describes the inputs in the schema.
func (a *RoomMemberArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.RoomID, "ID of the channel room, e.g. the ID of a Room resource.")
	an.Describe(&a.UserID, "User ID of the member, e.g. the ID of a Bot resource.")
}

// Create adds the member.
func (RoomMember) Create(
	ctx context.Context, req infer.CreateRequest[RoomMemberArgs],
) (infer.CreateResponse[RoomMemberState], error) {
	id := roomMemberID(req.Inputs)
	state := RoomMemberState{req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[RoomMemberState]{ID: id, Output: state}, nil
	}

	_, err := clientFrom(ctx).rooms.AddMember(ctx, connect.NewRequest(&apiv1.AddMemberRequest{
		RoomId: req.Inputs.RoomID,
		UserId: req.Inputs.UserID,
	}))
	// Adopt a membership that already exists, e.g. one added in the Chatto UI.
	if connect.CodeOf(err) == connect.CodeAlreadyExists {
		err = nil
	}

	return infer.CreateResponse[RoomMemberState]{ID: id, Output: state}, err
}

// Read refreshes the membership from Chatto. It also serves imports by resource ID. A user
// who isn't a member reads as deleted.
func (RoomMember) Read(
	ctx context.Context, req infer.ReadRequest[RoomMemberArgs, RoomMemberState],
) (infer.ReadResponse[RoomMemberArgs, RoomMemberState], error) {
	args, err := parseRoomMemberID(req.ID)
	if err != nil {
		return infer.ReadResponse[RoomMemberArgs, RoomMemberState]{}, err
	}

	_, err = clientFrom(ctx).rooms.GetMember(ctx, connect.NewRequest(&apiv1.GetMemberRequest{
		RoomId: args.RoomID,
		UserId: args.UserID,
	}))
	if isNotFound(err) {
		return infer.ReadResponse[RoomMemberArgs, RoomMemberState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[RoomMemberArgs, RoomMemberState]{}, err
	}

	return infer.ReadResponse[RoomMemberArgs, RoomMemberState]{
		ID:     req.ID,
		Inputs: args,
		State:  RoomMemberState{args},
	}, nil
}

// Delete removes the member.
func (RoomMember) Delete(ctx context.Context, req infer.DeleteRequest[RoomMemberState]) (infer.DeleteResponse, error) {
	_, err := clientFrom(ctx).rooms.RemoveMember(ctx, connect.NewRequest(&apiv1.RemoveMemberRequest{
		RoomId: req.State.RoomID,
		UserId: req.State.UserID,
	}))

	return infer.DeleteResponse{}, ignoreNotFound(err)
}

func roomMemberID(args RoomMemberArgs) string {
	return args.RoomID + "/" + args.UserID
}

func parseRoomMemberID(id string) (RoomMemberArgs, error) {
	roomID, userID, ok := strings.Cut(id, "/")
	if !ok || roomID == "" || userID == "" || strings.Contains(userID, "/") {
		return RoomMemberArgs{}, fmt.Errorf("invalid room member ID %q, expected <roomId>/<userId>", id)
	}

	return RoomMemberArgs{RoomID: roomID, UserID: userID}, nil
}
