package provider

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pulumi/pulumi-go-provider/infer"

	apiv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/api/v1"
)

// GetViewer looks up the account that the provider's API key belongs to.
type GetViewer struct{}

// GetViewerArgs are the inputs of GetViewer. It takes none.
type GetViewerArgs struct{}

// GetViewerResult is the account that the provider's API key belongs to.
type GetViewerResult struct {
	Bot         bool   `pulumi:"bot"`
	DisplayName string `pulumi:"displayName"`
	ID          string `pulumi:"id"`
	Login       string `pulumi:"login"`
}

// Annotate describes the function in the schema.
func (g *GetViewer) Annotate(a infer.Annotator) {
	a.Describe(&g, "Looks up the account that the provider's API key belongs to, e.g. to grant the "+
		"provider's own bot permissions with BotPermission.")
	a.SetToken("index", "getViewer")
}

// Annotate describes the outputs in the schema.
func (r *GetViewerResult) Annotate(an infer.Annotator) {
	an.Describe(&r.Bot, "Whether the account is a bot.")
	an.Describe(&r.DisplayName, "Display name.")
	an.Describe(&r.ID, "User ID.")
	an.Describe(&r.Login, "Username.")
}

// Invoke fetches the viewer.
func (GetViewer) Invoke(
	ctx context.Context, _ infer.FunctionRequest[GetViewerArgs],
) (infer.FunctionResponse[GetViewerResult], error) {
	res, err := clientFrom(ctx).viewer.GetViewer(ctx, connect.NewRequest(&apiv1.GetViewerRequest{}))
	if err != nil {
		return infer.FunctionResponse[GetViewerResult]{}, err
	}

	profile := res.Msg.GetUser().GetProfile()
	return infer.FunctionResponse[GetViewerResult]{Output: GetViewerResult{
		Bot:         profile.GetBot() != nil,
		DisplayName: profile.GetDisplayName(),
		ID:          profile.GetId(),
		Login:       profile.GetLogin(),
	}}, nil
}
