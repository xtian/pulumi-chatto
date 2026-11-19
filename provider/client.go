package provider

import (
	"context"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/xtian/pulumi-chatto/internal/pb/chatto/admin/v1/adminv1connect"
	"github.com/xtian/pulumi-chatto/internal/pb/chatto/api/v1/apiv1connect"
)

// client holds typed ConnectRPC clients for the Chatto services that the provider uses.
type client struct {
	bots        apiv1connect.BotServiceClient
	layout      adminv1connect.AdminRoomLayoutServiceClient
	permissions adminv1connect.AdminPermissionServiceClient
	roles       adminv1connect.AdminRoleServiceClient
	rooms       apiv1connect.RoomServiceClient
	users       apiv1connect.UserServiceClient
	viewer      apiv1connect.ViewerServiceClient
}

func newClient(httpClient *http.Client, baseURL, apiKey string) *client {
	url := strings.TrimSuffix(baseURL, "/") + "/api/connect"
	options := connect.WithInterceptors(bearerAuth(apiKey))

	return &client{
		bots:        apiv1connect.NewBotServiceClient(httpClient, url, options),
		layout:      adminv1connect.NewAdminRoomLayoutServiceClient(httpClient, url, options),
		permissions: adminv1connect.NewAdminPermissionServiceClient(httpClient, url, options),
		roles:       adminv1connect.NewAdminRoleServiceClient(httpClient, url, options),
		rooms:       apiv1connect.NewRoomServiceClient(httpClient, url, options),
		users:       apiv1connect.NewUserServiceClient(httpClient, url, options),
		viewer:      apiv1connect.NewViewerServiceClient(httpClient, url, options),
	}
}

func bearerAuth(apiKey string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+apiKey)
			return next(ctx, req)
		}
	}
}

func isNotFound(err error) bool {
	return connect.CodeOf(err) == connect.CodeNotFound
}

// ignoreNotFound treats a missing resource as already deleted.
func ignoreNotFound(err error) error {
	if isNotFound(err) {
		return nil
	}
	return err
}
