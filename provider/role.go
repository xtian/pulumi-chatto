package provider

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pulumi/pulumi-go-provider/infer"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	adminv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/admin/v1"
	apiv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/api/v1"
)

// Role manages a custom server role. The resource ID is the role name. Manage the
// permissions of a role with RolePermission.
type Role struct{}

// RoleArgs are the inputs of a Role.
type RoleArgs struct {
	Description        *string `pulumi:"description,optional"`
	DisplayName        *string `pulumi:"displayName,optional"`
	Name               string  `pulumi:"name"                 provider:"replaceOnChanges"`
	Pingable           *bool   `pulumi:"pingable,optional"`
	RequiresShieldMode *bool   `pulumi:"requiresShieldMode,optional"`
}

// RoleState is the stored state of a Role.
type RoleState struct {
	RoleArgs
	Position int `pulumi:"position"`
}

// Annotate describes the resource in the schema.
func (r *Role) Annotate(a infer.Annotator) {
	a.Describe(&r, "A custom Chatto role. The resource ID is the role name.")
}

// Annotate describes the inputs in the schema.
func (a *RoleArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Description, "Role description, up to 500 characters.")
	an.SetDefault(&a.Description, "")
	an.Describe(&a.DisplayName, "Name shown in admin and mention UIs, up to 80 characters. Defaults to `name`.")
	an.Describe(&a.Name, "Stable role name, e.g. the name an OIDC group maps to. Changing it replaces the role.")
	an.Describe(&a.Pingable, "Whether messages can mention the role to notify its members.")
	an.SetDefault(&a.Pingable, false)
	an.Describe(&a.RequiresShieldMode, "Whether members must select the role in Shield Mode before its grants apply.")
	an.SetDefault(&a.RequiresShieldMode, false)
}

// Annotate describes the outputs in the schema.
func (s *RoleState) Annotate(an infer.Annotator) {
	an.Describe(&s.Position, "Position in the role hierarchy.")
}

// Check applies the default display name, which is the role name.
func (Role) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[RoleArgs], error) {
	args, failures, err := infer.DefaultCheck[RoleArgs](ctx, req.NewInputs)
	if args.DisplayName == nil {
		args.DisplayName = &args.Name
	}

	return infer.CheckResponse[RoleArgs]{Inputs: args, Failures: failures}, err
}

// Create creates the role.
func (Role) Create(ctx context.Context, req infer.CreateRequest[RoleArgs]) (infer.CreateResponse[RoleState], error) {
	if req.DryRun {
		return infer.CreateResponse[RoleState]{ID: req.Inputs.Name, Output: RoleState{RoleArgs: req.Inputs}}, nil
	}

	res, err := clientFrom(ctx).roles.CreateRole(ctx, connect.NewRequest(&adminv1.CreateRoleRequest{
		Description:        deref(req.Inputs.Description),
		DisplayName:        deref(req.Inputs.DisplayName),
		Name:               req.Inputs.Name,
		Pingable:           deref(req.Inputs.Pingable),
		RequiresShieldMode: deref(req.Inputs.RequiresShieldMode),
	}))
	if err != nil {
		return infer.CreateResponse[RoleState]{}, err
	}

	return infer.CreateResponse[RoleState]{
		ID:     req.Inputs.Name,
		Output: RoleState{RoleArgs: req.Inputs, Position: int(res.Msg.GetRole().GetRole().GetPosition())},
	}, nil
}

// Read refreshes the role from Chatto. It also serves imports by role name.
func (Role) Read(
	ctx context.Context, req infer.ReadRequest[RoleArgs, RoleState],
) (infer.ReadResponse[RoleArgs, RoleState], error) {
	res, err := clientFrom(ctx).roles.GetRole(ctx, connect.NewRequest(&adminv1.GetRoleRequest{Name: req.ID}))
	if isNotFound(err) {
		return infer.ReadResponse[RoleArgs, RoleState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[RoleArgs, RoleState]{}, err
	}

	args, position := roleArgs(res.Msg.GetRole().GetRole())
	return infer.ReadResponse[RoleArgs, RoleState]{
		ID:     req.ID,
		Inputs: args,
		State:  RoleState{RoleArgs: args, Position: position},
	}, nil
}

// Update changes the role metadata.
func (Role) Update(
	ctx context.Context, req infer.UpdateRequest[RoleArgs, RoleState],
) (infer.UpdateResponse[RoleState], error) {
	state := RoleState{RoleArgs: req.Inputs, Position: req.State.Position}
	if req.DryRun {
		return infer.UpdateResponse[RoleState]{Output: state}, nil
	}

	description := deref(req.Inputs.Description)
	displayName := deref(req.Inputs.DisplayName)
	pingable := deref(req.Inputs.Pingable)
	requiresShieldMode := deref(req.Inputs.RequiresShieldMode)
	_, err := clientFrom(ctx).roles.UpdateRole(ctx, connect.NewRequest(&adminv1.UpdateRoleRequest{
		Description:        &description,
		DisplayName:        &displayName,
		Name:               req.ID,
		Pingable:           &pingable,
		RequiresShieldMode: &requiresShieldMode,
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"description", "display_name", "pingable", "requires_shield_mode"},
		},
	}))

	return infer.UpdateResponse[RoleState]{Output: state}, err
}

// Delete deletes the role.
func (Role) Delete(ctx context.Context, req infer.DeleteRequest[RoleState]) (infer.DeleteResponse, error) {
	_, err := clientFrom(ctx).roles.DeleteRole(ctx, connect.NewRequest(&adminv1.DeleteRoleRequest{Name: req.ID}))
	return infer.DeleteResponse{}, ignoreNotFound(err)
}

func roleArgs(role *apiv1.Role) (RoleArgs, int) {
	description := role.GetDescription()
	displayName := role.GetDisplayName()
	pingable := role.GetPingable()
	requiresShieldMode := role.GetRequiresShieldMode()

	return RoleArgs{
		Description:        &description,
		DisplayName:        &displayName,
		Name:               role.GetName(),
		Pingable:           &pingable,
		RequiresShieldMode: &requiresShieldMode,
	}, int(role.GetPosition())
}
