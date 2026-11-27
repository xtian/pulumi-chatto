package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	adminv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/admin/v1"
	"github.com/xtian/pulumi-chatto/internal/pb/chatto/admin/v1/adminv1connect"
	apiv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/api/v1"
)

// fakeRoles answers GetRole for one known role and reports every other role as missing.
type fakeRoles struct {
	adminv1connect.UnimplementedAdminRoleServiceHandler
	authorization string
	userAgent     string
}

func (f *fakeRoles) GetRole(
	_ context.Context, req *connect.Request[adminv1.GetRoleRequest],
) (*connect.Response[adminv1.GetRoleResponse], error) {
	f.authorization = req.Header().Get("Authorization")
	f.userAgent = req.Header().Get("User-Agent")
	if req.Msg.GetName() != "tech" {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&adminv1.GetRoleResponse{
		Role: &adminv1.AdminRole{Role: &apiv1.Role{Name: "tech", Position: 3}},
	}), nil
}

func TestClient(t *testing.T) {
	fake := &fakeRoles{}
	path, handler := adminv1connect.NewAdminRoleServiceHandler(fake)
	mux := http.NewServeMux()
	mux.Handle("/api/connect"+path, http.StripPrefix("/api/connect", handler))
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newClient(server.Client(), server.URL+"/", "test-key", "pulumi-chatto/1.2.3")

	tests := []struct {
		name         string
		role         string
		wantNotFound bool
		wantPosition int32
	}{
		{name: "existing role", role: "tech", wantPosition: 3},
		{name: "missing role", role: "nope", wantNotFound: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := c.roles.GetRole(context.Background(), connect.NewRequest(&adminv1.GetRoleRequest{Name: tt.role}))

			if got, want := fake.authorization, "Bearer test-key"; got != want {
				t.Fatalf("Authorization = %q, want %q", got, want)
			}
			if got, want := fake.userAgent, "pulumi-chatto/1.2.3"; got != want {
				t.Fatalf("User-Agent = %q, want %q", got, want)
			}
			if got := isNotFound(err); got != tt.wantNotFound {
				t.Fatalf("isNotFound(%v) = %v, want %v", err, got, tt.wantNotFound)
			}
			if ignoreNotFound(err) != nil {
				t.Fatalf("ignoreNotFound(%v) = %v, want nil", err, ignoreNotFound(err))
			}
			if !tt.wantNotFound && res.Msg.GetRole().GetRole().GetPosition() != tt.wantPosition {
				t.Fatalf("position = %d, want %d", res.Msg.GetRole().GetRole().GetPosition(), tt.wantPosition)
			}
		})
	}
}
