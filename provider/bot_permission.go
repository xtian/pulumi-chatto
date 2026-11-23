package provider

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pulumi/pulumi-go-provider/infer"

	adminv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/admin/v1"
)

// BotPermission grants one permission to a bot at a scope. Bots get no permissions from
// roles, and Chatto allows only explicit allows for them. The resource ID is
// <botUserId>/<permission>/<scopeKind>[/<scopeId>].
type BotPermission struct{}

// BotPermissionArgs are the inputs of a BotPermission.
type BotPermissionArgs struct {
	BotUserID  string     `pulumi:"botUserId"          provider:"replaceOnChanges"`
	Permission Permission `pulumi:"permission"         provider:"replaceOnChanges"`
	ScopeID    *string    `pulumi:"scopeId,optional"   provider:"replaceOnChanges"`
	ScopeKind  *ScopeKind `pulumi:"scopeKind,optional" provider:"replaceOnChanges"`
}

// BotPermissionState is the stored state of a BotPermission.
type BotPermissionState struct {
	BotPermissionArgs
}

// Annotate describes the resource in the schema.
func (b *BotPermission) Annotate(a infer.Annotator) {
	a.Describe(&b, "Grants one permission to a bot at server, group, room, or DM scope. A bot can use a "+
		"permission only while its owner also holds it. Deleting the resource removes the grant. Import with "+
		"`<botUserId>/<permission>/server`, `<botUserId>/<permission>/dm`, "+
		"`<botUserId>/<permission>/group/<groupId>`, or `<botUserId>/<permission>/room/<roomId>`.")
}

// Annotate describes the inputs in the schema.
func (a *BotPermissionArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.BotUserID, "User ID of the bot, e.g. the ID of a Bot resource.")
	an.Describe(&a.Permission, "Permission identifier. Each permission can be set only at the scopes it lists.")
	an.Describe(&a.ScopeID, "Room group ID for `group` scope, or room ID for `room` scope.")
	an.Describe(&a.ScopeKind, "Where the grant applies.")
	an.SetDefault(&a.ScopeKind, ScopeKindServer)
}

// Check validates the scope and rejects permissions Chatto never delegates to bots.
func (BotPermission) Check(
	ctx context.Context, req infer.CheckRequest,
) (infer.CheckResponse[BotPermissionArgs], error) {
	args, failures, err := infer.DefaultCheck[BotPermissionArgs](ctx, req.NewInputs)
	if err != nil || len(failures) > 0 {
		return infer.CheckResponse[BotPermissionArgs]{Inputs: args, Failures: failures}, err
	}

	failures = checkScope(args.Permission, args.ScopeKind, args.ScopeID, req.NewInputs.Get("scopeId").IsComputed())
	if args.Permission == PermissionUserDeleteSelf {
		failures = append(failures, checkFailure("permission", "Chatto doesn't delegate user.delete-self to bots"))
	}

	return infer.CheckResponse[BotPermissionArgs]{Inputs: args, Failures: failures}, nil
}

// Create grants the permission.
func (BotPermission) Create(
	ctx context.Context, req infer.CreateRequest[BotPermissionArgs],
) (infer.CreateResponse[BotPermissionState], error) {
	id := botPermissionID(req.Inputs)
	state := BotPermissionState{req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[BotPermissionState]{ID: id, Output: state}, nil
	}

	err := setBotPermission(ctx, req.Inputs, adminv1.PermissionDecision_PERMISSION_DECISION_ALLOW)
	return infer.CreateResponse[BotPermissionState]{ID: id, Output: state}, err
}

// Read refreshes the grant from Chatto. It also serves imports by resource ID. A bot
// without an explicit allow for the permission reads as deleted.
func (BotPermission) Read(
	ctx context.Context, req infer.ReadRequest[BotPermissionArgs, BotPermissionState],
) (infer.ReadResponse[BotPermissionArgs, BotPermissionState], error) {
	args, err := parseBotPermissionID(req.ID)
	if err != nil {
		return infer.ReadResponse[BotPermissionArgs, BotPermissionState]{}, err
	}

	res, err := clientFrom(ctx).permissions.ListUserPermissionDecisions(ctx,
		connect.NewRequest(&adminv1.ListUserPermissionDecisionsRequest{
			IncludeDirectMessageScope: resolvedScopeKind(args.ScopeKind) == ScopeKindDM,
			Scope:                     permissionScope(args.ScopeKind, args.ScopeID),
			UserId:                    args.BotUserID,
		}))
	if isNotFound(err) {
		return infer.ReadResponse[BotPermissionArgs, BotPermissionState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[BotPermissionArgs, BotPermissionState]{}, err
	}

	decision, ok := findDecision(res.Msg.GetDecisions(), args.Permission, args.ScopeKind, args.ScopeID)
	if !ok || decision != DecisionAllow {
		return infer.ReadResponse[BotPermissionArgs, BotPermissionState]{}, nil
	}

	return infer.ReadResponse[BotPermissionArgs, BotPermissionState]{
		ID:     req.ID,
		Inputs: args,
		State:  BotPermissionState{args},
	}, nil
}

// Delete removes the grant.
func (BotPermission) Delete(
	ctx context.Context, req infer.DeleteRequest[BotPermissionState],
) (infer.DeleteResponse, error) {
	err := setBotPermission(ctx, req.State.BotPermissionArgs, adminv1.PermissionDecision_PERMISSION_DECISION_NONE)
	return infer.DeleteResponse{}, ignoreNotFound(err)
}

func setBotPermission(ctx context.Context, args BotPermissionArgs, decision adminv1.PermissionDecision) error {
	_, err := clientFrom(ctx).permissions.SetUserPermission(ctx, connect.NewRequest(&adminv1.SetUserPermissionRequest{
		Decision:   decision,
		Permission: string(args.Permission),
		Scope:      permissionScope(args.ScopeKind, args.ScopeID),
		UserId:     args.BotUserID,
	}))
	return err
}

func botPermissionID(args BotPermissionArgs) string {
	return scopedPermissionID(scopedPermission{
		Permission: args.Permission,
		ScopeID:    args.ScopeID,
		ScopeKind:  args.ScopeKind,
		Subject:    args.BotUserID,
	})
}

func parseBotPermissionID(id string) (BotPermissionArgs, error) {
	sp, err := parseScopedPermissionID(id, "bot permission", "botUserId")
	if err != nil {
		return BotPermissionArgs{}, err
	}

	return BotPermissionArgs{
		BotUserID:  sp.Subject,
		Permission: sp.Permission,
		ScopeID:    sp.ScopeID,
		ScopeKind:  sp.ScopeKind,
	}, nil
}
