package provider

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pulumi/pulumi-go-provider/infer"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	adminv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/admin/v1"
	apiv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/api/v1"
)

// Room manages a channel room. The resource ID is the Chatto room ID. Chatto can't
// delete rooms, so Delete archives the room.
type Room struct{}

// ThreadingMode controls how a room uses threads.
type ThreadingMode string

// Threading modes of a room.
const (
	ThreadingModeEnabled    ThreadingMode = "enabled"
	ThreadingModeEncouraged ThreadingMode = "encouraged"
	ThreadingModeRequired   ThreadingMode = "required"
)

// Values lists the allowed threading modes for the schema.
func (ThreadingMode) Values() []infer.EnumValue[ThreadingMode] {
	return []infer.EnumValue[ThreadingMode]{
		{Name: "Enabled", Value: ThreadingModeEnabled, Description: "Replies may be threaded or inline."},
		{Name: "Encouraged", Value: ThreadingModeEncouraged, Description: "Clients steer replies into threads."},
		{Name: "Required", Value: ThreadingModeRequired, Description: "Every root message starts a thread."},
	}
}

var threadingModes = map[ThreadingMode]apiv1.RoomThreadingMode{
	ThreadingModeEnabled:    apiv1.RoomThreadingMode_ROOM_THREADING_MODE_ENABLED,
	ThreadingModeEncouraged: apiv1.RoomThreadingMode_ROOM_THREADING_MODE_ENCOURAGED,
	ThreadingModeRequired:   apiv1.RoomThreadingMode_ROOM_THREADING_MODE_REQUIRED,
}

// RoomArgs are the inputs of a Room.
type RoomArgs struct {
	Description     *string        `pulumi:"description,optional"`
	GroupID         string         `pulumi:"groupId"`
	Name            string         `pulumi:"name"`
	SlowModeSeconds *int           `pulumi:"slowModeSeconds,optional"`
	ThreadingMode   *ThreadingMode `pulumi:"threadingMode,optional"`
	Universal       *bool          `pulumi:"universal,optional"`
}

// RoomState is the stored state of a Room.
type RoomState struct {
	RoomArgs
}

// Annotate describes the resource in the schema.
func (r *Room) Annotate(a infer.Annotator) {
	a.Describe(&r, "A Chatto channel room. Chatto can't delete rooms, so deleting this resource archives the room. "+
		"Room names are unique across the server, including archived rooms.")
}

// Annotate describes the inputs in the schema.
func (a *RoomArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Description, "Room description, up to 500 characters.")
	an.SetDefault(&a.Description, "")
	an.Describe(&a.GroupID, "ID of the room group that contains the room. Changing it moves the room.")
	an.Describe(&a.Name, "Room name, 1-30 characters.")
	an.Describe(&a.SlowModeSeconds, "Minimum seconds between a member's root messages, up to 21600. 0 disables slow mode.")
	an.SetDefault(&a.SlowModeSeconds, 0)
	an.Describe(&a.ThreadingMode, "How the room uses threads.")
	an.SetDefault(&a.ThreadingMode, ThreadingModeEnabled)
	an.Describe(&a.Universal, "Whether every eligible server member is a member of the room.")
	an.SetDefault(&a.Universal, false)
}

// Create creates the room and then sets slow mode, which CreateRoom does not accept.
func (Room) Create(ctx context.Context, req infer.CreateRequest[RoomArgs]) (infer.CreateResponse[RoomState], error) {
	state := RoomState{req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[RoomState]{Output: state}, nil
	}

	c := clientFrom(ctx)
	res, err := c.rooms.CreateRoom(ctx, connect.NewRequest(&apiv1.CreateRoomRequest{
		Description:   deref(req.Inputs.Description),
		GroupId:       req.Inputs.GroupID,
		Name:          req.Inputs.Name,
		ThreadingMode: threadingModes[threadingMode(req.Inputs)],
		Universal:     deref(req.Inputs.Universal),
	}))
	if err != nil {
		return infer.CreateResponse[RoomState]{}, err
	}

	id := res.Msg.GetRoom().GetId()
	if seconds := uint32(deref(req.Inputs.SlowModeSeconds)); seconds != 0 {
		_, err = c.rooms.UpdateRoom(ctx, connect.NewRequest(&apiv1.UpdateRoomRequest{
			RoomId:          id,
			SlowModeSeconds: &seconds,
			UpdateMask:      &fieldmaskpb.FieldMask{Paths: []string{"slow_mode_seconds"}},
		}))
	}

	return infer.CreateResponse[RoomState]{ID: id, Output: state}, err
}

// Read refreshes the room from Chatto. An archived room reads as deleted.
func (Room) Read(
	ctx context.Context, req infer.ReadRequest[RoomArgs, RoomState],
) (infer.ReadResponse[RoomArgs, RoomState], error) {
	res, err := clientFrom(ctx).layout.GetRoom(ctx, connect.NewRequest(&adminv1.GetRoomRequest{RoomId: req.ID}))
	if isNotFound(err) || (err == nil && res.Msg.GetRoom().GetArchived()) {
		return infer.ReadResponse[RoomArgs, RoomState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[RoomArgs, RoomState]{}, err
	}

	room := res.Msg.GetRoom()
	mode := ThreadingModeEnabled
	for candidate, value := range threadingModes {
		if value == room.GetThreadingMode() {
			mode = candidate
		}
	}
	description := room.GetDescription()
	slowModeSeconds := int(room.GetSlowModeSeconds())
	universal := room.GetUniversal()

	args := RoomArgs{
		Description:     &description,
		GroupID:         room.GetGroupId(),
		Name:            room.GetName(),
		SlowModeSeconds: &slowModeSeconds,
		ThreadingMode:   &mode,
		Universal:       &universal,
	}
	return infer.ReadResponse[RoomArgs, RoomState]{ID: req.ID, Inputs: args, State: RoomState{args}}, nil
}

// Update moves the room when its group changes, then updates its settings.
func (Room) Update(
	ctx context.Context, req infer.UpdateRequest[RoomArgs, RoomState],
) (infer.UpdateResponse[RoomState], error) {
	state := RoomState{req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[RoomState]{Output: state}, nil
	}

	c := clientFrom(ctx)
	if req.State.GroupID != req.Inputs.GroupID {
		_, err := c.layout.MoveRoomToGroup(ctx, connect.NewRequest(&adminv1.MoveRoomToGroupRequest{
			GroupId: req.Inputs.GroupID,
			RoomId:  req.ID,
		}))
		if err != nil {
			return infer.UpdateResponse[RoomState]{}, err
		}
	}

	description := deref(req.Inputs.Description)
	mode := threadingModes[threadingMode(req.Inputs)]
	slowModeSeconds := uint32(deref(req.Inputs.SlowModeSeconds))
	universal := deref(req.Inputs.Universal)
	_, err := c.rooms.UpdateRoom(ctx, connect.NewRequest(&apiv1.UpdateRoomRequest{
		Description:     &description,
		Name:            &req.Inputs.Name,
		RoomId:          req.ID,
		SlowModeSeconds: &slowModeSeconds,
		ThreadingMode:   &mode,
		Universal:       &universal,
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"description", "name", "slow_mode_seconds", "threading_mode", "universal"},
		},
	}))

	return infer.UpdateResponse[RoomState]{Output: state}, err
}

// Delete archives the room.
func (Room) Delete(ctx context.Context, req infer.DeleteRequest[RoomState]) (infer.DeleteResponse, error) {
	_, err := clientFrom(ctx).rooms.ArchiveRoom(ctx, connect.NewRequest(&apiv1.ArchiveRoomRequest{RoomId: req.ID}))
	return infer.DeleteResponse{}, ignoreNotFound(err)
}

func threadingMode(args RoomArgs) ThreadingMode {
	if args.ThreadingMode == nil {
		return ThreadingModeEnabled
	}
	return *args.ThreadingMode
}
