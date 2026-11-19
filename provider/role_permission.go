package provider

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pulumi/pulumi-go-provider/infer"

	adminv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/admin/v1"
)

// RolePermission manages one explicit allow or deny decision for a role at a scope.
// The resource ID is <roleName>/<permission>/<scopeKind>[/<scopeId>].
type RolePermission struct{}

// Permission is a Chatto permission identifier. permissions_gen.go defines the values.
type Permission string

// Decision is an explicit permission decision.
type Decision string

// Permission decisions.
const (
	DecisionAllow Decision = "allow"
	DecisionDeny  Decision = "deny"
)

// Values lists the allowed decisions for the schema.
func (Decision) Values() []infer.EnumValue[Decision] {
	return []infer.EnumValue[Decision]{
		{Name: "Allow", Value: DecisionAllow, Description: "Grant the permission."},
		{Name: "Deny", Value: DecisionDeny, Description: "Deny the permission, overriding broader grants."},
	}
}

var decisions = map[Decision]adminv1.PermissionDecision{
	DecisionAllow: adminv1.PermissionDecision_PERMISSION_DECISION_ALLOW,
	DecisionDeny:  adminv1.PermissionDecision_PERMISSION_DECISION_DENY,
}

// ScopeKind is the level at which a permission decision applies.
type ScopeKind string

// Permission scopes.
const (
	ScopeKindDM     ScopeKind = "dm"
	ScopeKindGroup  ScopeKind = "group"
	ScopeKindRoom   ScopeKind = "room"
	ScopeKindServer ScopeKind = "server"
)

// Values lists the allowed scope kinds for the schema.
func (ScopeKind) Values() []infer.EnumValue[ScopeKind] {
	return []infer.EnumValue[ScopeKind]{
		{Name: "DM", Value: ScopeKindDM, Description: "Applies in direct-message rooms."},
		{Name: "Group", Value: ScopeKindGroup, Description: "Applies within one room group."},
		{Name: "Room", Value: ScopeKindRoom, Description: "Applies within one room."},
		{Name: "Server", Value: ScopeKindServer, Description: "Applies server-wide."},
	}
}

var scopeKinds = map[ScopeKind]adminv1.PermissionScopeKind{
	ScopeKindDM:     adminv1.PermissionScopeKind_PERMISSION_SCOPE_KIND_DM,
	ScopeKindGroup:  adminv1.PermissionScopeKind_PERMISSION_SCOPE_KIND_GROUP,
	ScopeKindRoom:   adminv1.PermissionScopeKind_PERMISSION_SCOPE_KIND_ROOM,
	ScopeKindServer: adminv1.PermissionScopeKind_PERMISSION_SCOPE_KIND_SERVER,
}

// RolePermissionArgs are the inputs of a RolePermission.
type RolePermissionArgs struct {
	Decision   Decision   `pulumi:"decision"`
	Permission Permission `pulumi:"permission"         provider:"replaceOnChanges"`
	RoleName   string     `pulumi:"roleName"           provider:"replaceOnChanges"`
	ScopeID    *string    `pulumi:"scopeId,optional"   provider:"replaceOnChanges"`
	ScopeKind  *ScopeKind `pulumi:"scopeKind,optional" provider:"replaceOnChanges"`
}

// RolePermissionState is the stored state of a RolePermission.
type RolePermissionState struct {
	RolePermissionArgs
}

// Annotate describes the resource in the schema.
func (r *RolePermission) Annotate(a infer.Annotator) {
	a.Describe(&r, "An explicit allow or deny of one permission for a role at server, group, room, or DM scope. "+
		"Deleting it clears the decision. Import with `<roleName>/<permission>/server`, `<roleName>/<permission>/dm`, "+
		"`<roleName>/<permission>/group/<groupId>`, or `<roleName>/<permission>/room/<roomId>`.")
}

// Annotate describes the inputs in the schema.
func (a *RolePermissionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.Decision, "Whether the role is allowed or denied the permission.")
	an.Describe(&a.Permission, "Permission identifier. Each permission can be set only at the scopes it lists.")
	an.Describe(&a.RoleName, "Name of the role, including system roles such as `everyone` and `moderator`.")
	an.Describe(&a.ScopeID, "Room group ID for `group` scope, or room ID for `room` scope.")
	an.Describe(&a.ScopeKind, "Where the decision applies.")
	an.SetDefault(&a.ScopeKind, ScopeKindServer)
}

// Check requires a scope ID exactly for group and room scopes, and requires a scope that
// the permission supports.
func (RolePermission) Check(
	ctx context.Context, req infer.CheckRequest,
) (infer.CheckResponse[RolePermissionArgs], error) {
	args, failures, err := infer.DefaultCheck[RolePermissionArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[RolePermissionArgs]{Inputs: args, Failures: failures}, err
	}

	failures = checkScope(args.Permission, args.ScopeKind, args.ScopeID)
	return infer.CheckResponse[RolePermissionArgs]{Inputs: args, Failures: failures}, nil
}

// Create sets the decision.
func (RolePermission) Create(
	ctx context.Context, req infer.CreateRequest[RolePermissionArgs],
) (infer.CreateResponse[RolePermissionState], error) {
	id := rolePermissionID(req.Inputs)
	state := RolePermissionState{req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[RolePermissionState]{ID: id, Output: state}, nil
	}

	err := setRolePermission(ctx, req.Inputs, decisions[req.Inputs.Decision])
	return infer.CreateResponse[RolePermissionState]{ID: id, Output: state}, err
}

// Read refreshes the decision from Chatto. It also serves imports by resource ID. A role
// without an explicit decision for the permission reads as deleted.
func (RolePermission) Read(
	ctx context.Context, req infer.ReadRequest[RolePermissionArgs, RolePermissionState],
) (infer.ReadResponse[RolePermissionArgs, RolePermissionState], error) {
	args, err := parseRolePermissionID(req.ID)
	if err != nil {
		return infer.ReadResponse[RolePermissionArgs, RolePermissionState]{}, err
	}

	res, err := clientFrom(ctx).permissions.ListRolePermissionDecisions(ctx,
		connect.NewRequest(&adminv1.ListRolePermissionDecisionsRequest{
			IncludeDirectMessageScope: resolvedScopeKind(args.ScopeKind) == ScopeKindDM,
			RoleName:                  args.RoleName,
			Scope:                     permissionScope(args.ScopeKind, args.ScopeID),
		}))
	if isNotFound(err) {
		return infer.ReadResponse[RolePermissionArgs, RolePermissionState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[RolePermissionArgs, RolePermissionState]{}, err
	}

	decision, ok := findDecision(res.Msg.GetDecisions(), args.Permission, args.ScopeKind, args.ScopeID)
	if !ok {
		return infer.ReadResponse[RolePermissionArgs, RolePermissionState]{}, nil
	}

	args.Decision = decision
	return infer.ReadResponse[RolePermissionArgs, RolePermissionState]{
		ID:     req.ID,
		Inputs: args,
		State:  RolePermissionState{args},
	}, nil
}

// Update changes the decision.
func (RolePermission) Update(
	ctx context.Context, req infer.UpdateRequest[RolePermissionArgs, RolePermissionState],
) (infer.UpdateResponse[RolePermissionState], error) {
	state := RolePermissionState{req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[RolePermissionState]{Output: state}, nil
	}

	err := setRolePermission(ctx, req.Inputs, decisions[req.Inputs.Decision])
	return infer.UpdateResponse[RolePermissionState]{Output: state}, err
}

// Delete clears the decision.
func (RolePermission) Delete(
	ctx context.Context, req infer.DeleteRequest[RolePermissionState],
) (infer.DeleteResponse, error) {
	err := setRolePermission(ctx, req.State.RolePermissionArgs, adminv1.PermissionDecision_PERMISSION_DECISION_NONE)
	return infer.DeleteResponse{}, ignoreNotFound(err)
}

func setRolePermission(ctx context.Context, args RolePermissionArgs, decision adminv1.PermissionDecision) error {
	_, err := clientFrom(ctx).permissions.SetRolePermission(ctx, connect.NewRequest(&adminv1.SetRolePermissionRequest{
		Decision:   decision,
		Permission: string(args.Permission),
		RoleName:   args.RoleName,
		Scope:      permissionScope(args.ScopeKind, args.ScopeID),
	}))
	return err
}

func rolePermissionID(args RolePermissionArgs) string {
	return scopedPermissionID(scopedPermission{
		Permission: args.Permission,
		ScopeID:    args.ScopeID,
		ScopeKind:  args.ScopeKind,
		Subject:    args.RoleName,
	})
}

func parseRolePermissionID(id string) (RolePermissionArgs, error) {
	sp, err := parseScopedPermissionID(id, "role permission", "roleName")
	if err != nil {
		return RolePermissionArgs{}, err
	}

	return RolePermissionArgs{
		Permission: sp.Permission,
		RoleName:   sp.Subject,
		ScopeID:    sp.ScopeID,
		ScopeKind:  sp.ScopeKind,
	}, nil
}
