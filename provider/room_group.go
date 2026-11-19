package provider

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pulumi/pulumi-go-provider/infer"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	adminv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/admin/v1"
)

// RoomGroup manages a room group: a sidebar section that holds channel rooms.
// The resource ID is the Chatto group ID.
type RoomGroup struct{}

// RoomGroupArgs are the inputs of a RoomGroup.
type RoomGroupArgs struct {
	Description *string `pulumi:"description,optional"`
	Name        string  `pulumi:"name"`
}

// RoomGroupState is the stored state of a RoomGroup.
type RoomGroupState struct {
	RoomGroupArgs
}

// Annotate describes the resource in the schema.
func (r *RoomGroup) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Chatto room group. Chatto only deletes empty groups, and archived rooms still count, "+
		"so a group can't be deleted once it has held a room.")
}

// Annotate describes the inputs in the schema.
func (a *RoomGroupArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Description, "Public group description.")
	an.SetDefault(&a.Description, "")
	an.Describe(&a.Name, "Group display name, up to 80 characters.")
}

// Create creates the room group.
func (RoomGroup) Create(
	ctx context.Context, req infer.CreateRequest[RoomGroupArgs],
) (infer.CreateResponse[RoomGroupState], error) {
	state := RoomGroupState{req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[RoomGroupState]{Output: state}, nil
	}

	res, err := clientFrom(ctx).layout.CreateRoomGroup(ctx, connect.NewRequest(&adminv1.CreateRoomGroupRequest{
		Description: deref(req.Inputs.Description),
		Name:        req.Inputs.Name,
	}))
	if err != nil {
		return infer.CreateResponse[RoomGroupState]{}, err
	}

	return infer.CreateResponse[RoomGroupState]{ID: res.Msg.GetGroup().GetId(), Output: state}, nil
}

// Read refreshes the room group from Chatto. It also serves imports by group ID.
func (RoomGroup) Read(
	ctx context.Context, req infer.ReadRequest[RoomGroupArgs, RoomGroupState],
) (infer.ReadResponse[RoomGroupArgs, RoomGroupState], error) {
	res, err := clientFrom(ctx).layout.GetRoomGroup(ctx, connect.NewRequest(&adminv1.GetRoomGroupRequest{
		GroupId: req.ID,
	}))
	if isNotFound(err) {
		return infer.ReadResponse[RoomGroupArgs, RoomGroupState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[RoomGroupArgs, RoomGroupState]{}, err
	}

	group := res.Msg.GetGroup()
	description := group.GetDescription()
	args := RoomGroupArgs{Description: &description, Name: group.GetName()}
	return infer.ReadResponse[RoomGroupArgs, RoomGroupState]{
		ID:     req.ID,
		Inputs: args,
		State:  RoomGroupState{args},
	}, nil
}

// Update changes the group name and description.
func (RoomGroup) Update(
	ctx context.Context, req infer.UpdateRequest[RoomGroupArgs, RoomGroupState],
) (infer.UpdateResponse[RoomGroupState], error) {
	state := RoomGroupState{req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[RoomGroupState]{Output: state}, nil
	}

	description := deref(req.Inputs.Description)
	_, err := clientFrom(ctx).layout.UpdateRoomGroup(ctx, connect.NewRequest(&adminv1.UpdateRoomGroupRequest{
		Description: &description,
		GroupId:     req.ID,
		Name:        &req.Inputs.Name,
		UpdateMask:  &fieldmaskpb.FieldMask{Paths: []string{"description", "name"}},
	}))

	return infer.UpdateResponse[RoomGroupState]{Output: state}, err
}

// Delete deletes the room group. Chatto refuses when the group still holds rooms, archived or not.
func (RoomGroup) Delete(ctx context.Context, req infer.DeleteRequest[RoomGroupState]) (infer.DeleteResponse, error) {
	_, err := clientFrom(ctx).layout.DeleteRoomGroup(ctx, connect.NewRequest(&adminv1.DeleteRoomGroupRequest{
		GroupId: req.ID,
	}))

	return infer.DeleteResponse{}, ignoreNotFound(err)
}

func deref[T any](value *T) T {
	if value == nil {
		var zero T
		return zero
	}
	return *value
}
