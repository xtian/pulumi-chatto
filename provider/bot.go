package provider

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pulumi/pulumi-go-provider/infer"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	apiv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/api/v1"
)

// Bot manages a bot account. The resource ID is the bot's user ID. Grant it permissions
// with BotPermission.
type Bot struct{}

// BotArgs are the inputs of a Bot.
type BotArgs struct {
	DisplayName *string `pulumi:"displayName,optional"`
	Login       string  `pulumi:"login"`
}

// BotState is the stored state of a Bot.
type BotState struct {
	BotArgs
	APIKey      *string `pulumi:"apiKey,optional" provider:"secret"`
	OwnerUserID string  `pulumi:"ownerUserId"`
}

// Annotate describes the resource in the schema.
func (b *Bot) Annotate(a infer.Annotator) {
	a.Describe(&b, "A Chatto bot account. The resource ID is the bot's user ID. Chatto makes the human owner of "+
		"the provider's API key the owner of the bot. Deleting the resource deletes the bot.")
}

// Annotate describes the inputs in the schema.
func (a *BotArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.DisplayName, "Public display name, up to 32 characters. Defaults to `login`.")
	an.Describe(&a.Login, "Bot username. Chatto allows one change every 30 days unless the provider's API key "+
		"has user.manage-accounts.")
}

// Annotate describes the outputs in the schema.
func (s *BotState) Annotate(an infer.Annotator) {
	an.Describe(&s.APIKey, "The API key Chatto issues when it creates the bot. Chatto shows it only once, so it "+
		"is unset on imported bots.")
	an.Describe(&s.OwnerUserID, "User ID of the bot's human owner. Bot permissions never exceed the owner's.")
}

// Check applies the default display name, which is the login.
func (Bot) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[BotArgs], error) {
	args, failures, err := infer.DefaultCheck[BotArgs](ctx, req.NewInputs)
	if args.DisplayName == nil {
		args.DisplayName = &args.Login
	}

	return infer.CheckResponse[BotArgs]{Inputs: args, Failures: failures}, err
}

// Create creates the bot and keeps its initial API key.
func (Bot) Create(ctx context.Context, req infer.CreateRequest[BotArgs]) (infer.CreateResponse[BotState], error) {
	if req.DryRun {
		return infer.CreateResponse[BotState]{Output: BotState{BotArgs: req.Inputs}}, nil
	}

	res, err := clientFrom(ctx).bots.CreateBot(ctx, connect.NewRequest(&apiv1.CreateBotRequest{
		DisplayName: deref(req.Inputs.DisplayName),
		Login:       req.Inputs.Login,
	}))
	if err != nil {
		return infer.CreateResponse[BotState]{}, err
	}

	bot := res.Msg.GetBot()
	apiKey := res.Msg.GetApiKey()
	return infer.CreateResponse[BotState]{
		ID:     bot.GetUser().GetId(),
		Output: BotState{APIKey: &apiKey, BotArgs: req.Inputs, OwnerUserID: bot.GetOwnerUserId()},
	}, nil
}

// Read refreshes the bot from Chatto. It also serves imports by user ID.
func (Bot) Read(
	ctx context.Context, req infer.ReadRequest[BotArgs, BotState],
) (infer.ReadResponse[BotArgs, BotState], error) {
	res, err := clientFrom(ctx).bots.GetBot(ctx, connect.NewRequest(&apiv1.GetBotRequest{BotUserId: req.ID}))
	if isNotFound(err) {
		return infer.ReadResponse[BotArgs, BotState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[BotArgs, BotState]{}, err
	}

	bot := res.Msg.GetBot()
	user := bot.GetUser()
	if user.GetDeleted() {
		return infer.ReadResponse[BotArgs, BotState]{}, nil
	}

	displayName := user.GetDisplayName()
	args := BotArgs{DisplayName: &displayName, Login: user.GetLogin()}
	return infer.ReadResponse[BotArgs, BotState]{
		ID:     req.ID,
		Inputs: args,
		State:  BotState{APIKey: req.State.APIKey, BotArgs: args, OwnerUserID: bot.GetOwnerUserId()},
	}, nil
}

// Update changes the login and display name. It sends only changed fields, since a login
// change starts Chatto's 30-day cooldown.
func (Bot) Update(ctx context.Context, req infer.UpdateRequest[BotArgs, BotState]) (infer.UpdateResponse[BotState], error) {
	state := BotState{APIKey: req.State.APIKey, BotArgs: req.Inputs, OwnerUserID: req.State.OwnerUserID}
	if req.DryRun {
		return infer.UpdateResponse[BotState]{Output: state}, nil
	}

	update := &apiv1.UpdateUserProfileRequest{UpdateMask: &fieldmaskpb.FieldMask{}, UserId: req.ID}
	if deref(req.Inputs.DisplayName) != deref(req.State.DisplayName) {
		update.DisplayName = req.Inputs.DisplayName
		update.UpdateMask.Paths = append(update.UpdateMask.Paths, "display_name")
	}
	if req.Inputs.Login != req.State.Login {
		update.Login = &req.Inputs.Login
		update.UpdateMask.Paths = append(update.UpdateMask.Paths, "login")
	}
	if len(update.UpdateMask.Paths) == 0 {
		return infer.UpdateResponse[BotState]{Output: state}, nil
	}

	_, err := clientFrom(ctx).users.UpdateUserProfile(ctx, connect.NewRequest(update))
	return infer.UpdateResponse[BotState]{Output: state}, err
}

// Delete deletes the bot, which revokes its API keys.
func (Bot) Delete(ctx context.Context, req infer.DeleteRequest[BotState]) (infer.DeleteResponse, error) {
	_, err := clientFrom(ctx).bots.DeleteBot(ctx, connect.NewRequest(&apiv1.DeleteBotRequest{BotUserId: req.ID}))
	return infer.DeleteResponse{}, ignoreNotFound(err)
}
