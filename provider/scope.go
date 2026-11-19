package provider

import (
	"fmt"
	"slices"
	"strings"

	p "github.com/pulumi/pulumi-go-provider"

	adminv1 "github.com/xtian/pulumi-chatto/internal/pb/chatto/admin/v1"
)

// scopedPermission identifies one permission at one scope for a subject (a role name or a user ID).
type scopedPermission struct {
	Permission Permission
	ScopeID    *string
	ScopeKind  *ScopeKind
	Subject    string
}

// checkScope requires a scope ID exactly for group and room scopes, and requires a scope that
// the permission supports.
func checkScope(permission Permission, kind *ScopeKind, scopeID *string) []p.CheckFailure {
	var failures []p.CheckFailure

	resolved := resolvedScopeKind(kind)
	needsID := resolved == ScopeKindGroup || resolved == ScopeKindRoom
	switch {
	case needsID && deref(scopeID) == "":
		failures = append(failures, checkFailure("scopeId", fmt.Sprintf("is required for %s scope", resolved)))
	case !needsID && deref(scopeID) != "":
		failures = append(failures, checkFailure("scopeId", fmt.Sprintf("must be unset for %s scope", resolved)))
	}

	if scopes, known := permissionScopes[permission]; known && !slices.Contains(scopes, resolved) {
		failures = append(failures, checkFailure("scopeKind",
			fmt.Sprintf("%s can't be set at %s scope; it supports %v", permission, resolved, scopes)))
	}

	return failures
}

func checkFailure(property, reason string) p.CheckFailure {
	return p.CheckFailure{Property: property, Reason: reason}
}

// findDecision returns the explicit decision for permission at exactly the given scope.
func findDecision(
	entries []*adminv1.ScopedPermissionDecision, permission Permission, kind *ScopeKind, scopeID *string,
) (Decision, bool) {
	for _, decision := range entries {
		scope := decision.GetScope()
		if decision.GetPermission() != string(permission) ||
			scope.GetKind() != scopeKinds[resolvedScopeKind(kind)] ||
			scope.GetId() != deref(scopeID) {
			continue
		}

		for candidate, value := range decisions {
			if value == decision.GetOverride() {
				return candidate, true
			}
		}
	}

	return "", false
}

func permissionScope(kind *ScopeKind, scopeID *string) *adminv1.PermissionScope {
	return &adminv1.PermissionScope{Id: deref(scopeID), Kind: scopeKinds[resolvedScopeKind(kind)]}
}

func resolvedScopeKind(kind *ScopeKind) ScopeKind {
	if kind == nil {
		return ScopeKindServer
	}
	return *kind
}

func scopedPermissionID(sp scopedPermission) string {
	parts := []string{sp.Subject, string(sp.Permission), string(resolvedScopeKind(sp.ScopeKind))}
	if id := deref(sp.ScopeID); id != "" {
		parts = append(parts, id)
	}
	return strings.Join(parts, "/")
}

// parseScopedPermissionID parses <subject>/<permission>/<scopeKind>[/<scopeId>]. subjectName
// names the first part in error messages.
func parseScopedPermissionID(id, resource, subjectName string) (scopedPermission, error) {
	parts := strings.Split(id, "/")
	if len(parts) < 3 || len(parts) > 4 {
		return scopedPermission{}, fmt.Errorf(
			"%s ID %q should look like <%s>/<permission>/<scopeKind>[/<scopeId>]", resource, id, subjectName)
	}

	kind := ScopeKind(parts[2])
	if _, ok := scopeKinds[kind]; !ok {
		return scopedPermission{}, fmt.Errorf("unknown scope kind %q in %s ID %q", parts[2], resource, id)
	}

	sp := scopedPermission{Permission: Permission(parts[1]), ScopeKind: &kind, Subject: parts[0]}
	if len(parts) == 4 {
		sp.ScopeID = &parts[3]
	}
	return sp, nil
}
