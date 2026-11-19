// Package provider implements a Pulumi provider for Chatto (https://chatto.run).
package provider

import (
	"context"
	"errors"
	"net/http"
	"time"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

// Name is the Pulumi package name of the provider.
const Name = "chatto"

// Provider builds the Chatto provider.
func Provider() (p.Provider, error) {
	return infer.NewProviderBuilder().
		WithConfig(infer.Config(&Config{})).
		WithDescription("Manage Chatto bots, bot permissions, rooms, room groups, room members, roles, and role permissions.").
		WithDisplayName("Chatto").
		WithGoImportPath("github.com/xtian/pulumi-chatto/sdk/go/chatto").
		WithHomepage("https://github.com/xtian/pulumi-chatto").
		WithKeywords("chatto", "chat", "category/utility").
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{"provider": "index"}).
		WithNamespace("xtian").
		WithPluginDownloadURL("github://api.github.com/xtian/pulumi-chatto").
		WithPublisher("xtian").
		WithRepository("https://github.com/xtian/pulumi-chatto").
		WithFunctions(infer.Function(&GetViewer{})).
		WithResources(
			infer.Resource(&Bot{}),
			infer.Resource(&BotPermission{}),
			infer.Resource(&Role{}),
			infer.Resource(&RolePermission{}),
			infer.Resource(&Room{}),
			infer.Resource(&RoomGroup{}),
			infer.Resource(&RoomMember{}),
		).
		Build()
}

// Config is the provider configuration. Each provider instance talks to one Chatto server.
type Config struct {
	APIKey *string `pulumi:"apiKey,optional" provider:"secret"`
	URL    *string `pulumi:"url,optional"`

	client *client
}

// Annotate describes the configuration fields in the schema.
func (c *Config) Annotate(a infer.Annotator) {
	a.Describe(&c.APIKey, "Bot API key. The bot needs explicit grants for each permission it uses "+
		"(room.create, room.manage, role.manage), which its owner must also hold. Can be set with CHATTO_API_KEY.")
	a.SetDefault(&c.APIKey, nil, "CHATTO_API_KEY")
	a.Describe(&c.URL, "Base URL of the Chatto server, e.g. https://chat.example.com. Can be set with CHATTO_URL.")
	a.SetDefault(&c.URL, nil, "CHATTO_URL")
}

// Configure validates the configuration and creates the API client.
func (c *Config) Configure(context.Context) error {
	switch {
	case deref(c.URL) == "":
		return errors.New("chatto:url (or CHATTO_URL) is required")
	case deref(c.APIKey) == "":
		return errors.New("chatto:apiKey (or CHATTO_API_KEY) is required")
	}

	c.client = newClient(&http.Client{Timeout: time.Minute}, *c.URL, *c.APIKey)
	return nil
}

func clientFrom(ctx context.Context) *client {
	return infer.GetConfig[*Config](ctx).client
}
