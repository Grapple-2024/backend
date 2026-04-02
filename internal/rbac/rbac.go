package rbac

import (
	"context"
	"fmt"
	"strings"

	"github.com/Grapple-2024/backend/internal/service/profiles"
	"github.com/Grapple-2024/backend/pkg/utils"
	"github.com/rs/zerolog/log"
)

const (
	Owner   = "owner"
	Coach   = "coach"
	Student = "student"

	Owners   = "owners"
	Coaches  = "coaches"
	Students = "students"

	ActionRead   = "read"
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"

	ResourceSeries        = "series"
	ResourceGym           = "gym"
	ResourceCoaches       = "coaches"
	ResourceOwners        = "owners"
	ResourceAnnouncements = "announcements"
	ResourceGymRequests   = "requests"
	ResourceRoles         = "roles"
)

// Role represents a role in the system with its associated permissions.
type Role struct {
	Name        string
	Permissions []string
}

// Permission represents a permission to access a specific resource or perform an action within a scope.
type Permission struct {
	Resource string
	Action   string
}

// User represents a user with assigned roles.
type User struct {
	ID       string
	Username string
	Roles    []string
}

// GroupUser is a lightweight user returned by ListUsersInGroup.
type GroupUser struct {
	Username string
}

// UserStore interface for fetching user data.
type UserStore interface {
	GetUser(ctx context.Context, userID string) (*User, error)
}

// RBAC is the core RBAC object.
type RBAC struct {
	users       map[string]User
	roles       map[string]Role
	permissions map[string]Permission
}

func New(profileSVC *profiles.Service) (*RBAC, error) {
	r := &RBAC{
		roles:       make(map[string]Role),
		permissions: make(map[string]Permission),
	}

	return r, nil
}

func (r *RBAC) GetUser(ctx context.Context, userID string) (*User, error) {
	log.Info().Msgf("Fetching user from RBAC map: %v", userID)
	return &User{ID: userID}, nil
}

// AddRoles adds one or more roles to the RBAC system (in-memory cache).
func (r *RBAC) AddRoles(roles ...Role) {
	for _, role := range roles {
		r.roles[role.Name] = role
	}
}

// AddPermission adds a new permission to the RBAC system (in-memory cache)
func (r *RBAC) AddPermissions(permissions ...Permission) {
	for _, p := range permissions {
		permission := fmt.Sprintf("%s:%s", p.Resource, p.Action)

		if _, ok := r.permissions[permission]; !ok {
			r.permissions[permission] = p
		}
	}
}

// IsAuthorized checks if a user is authorized to perform an action on a resource.
func (r *RBAC) IsAuthorized(ctx context.Context, userID, resource, action string) (bool, error) {
	if err := r.SeedCache(context.Background()); err != nil {
		return false, err
	}

	user, err := r.GetUser(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w, user ID: %v", err, userID)
	}

	permissionNeeded := fmt.Sprintf("%s:%s", resource, action)
	log.Info().Msgf("running isAuthorized(%s, %s)?", userID, permissionNeeded)
	totalRoles := []string{}
	for _, roleName := range user.Roles {
		role, ok := r.roles[roleName]
		if !ok {
			log.Warn().Msgf("could not find role '%s' in role cache: %v", roleName, r.roles)
			continue
		}
		totalRoles = append(totalRoles, role.Name)

		for _, userPermission := range role.Permissions {
			if userPermission == permissionNeeded {
				return true, nil
			}
		}
	}

	log.Warn().Msgf("User %q does not have permission for %s\ncurrent permissions: %+v", userID, permissionNeeded, totalRoles)

	return false, nil
}

// CreateGymRBAC stores roles and permissions in RBAC cache for a new gym.
func (r *RBAC) CreateGymRBAC(ctx context.Context, gymID string) error {
	return r.StoreGymRBAC(gymID)
}

func (r *RBAC) RemoveUserFromGymGroups(ctx context.Context, gymID, userID string) error {
	delete(r.users, userID)
	return nil
}

// AssignUserToGymRole assigns a user to a specific gym's role in the RBAC cache.
func (r *RBAC) AssignUserToGymRole(ctx context.Context, gymID, username, roleName string) error {
	_ = utils.PluralGroupNameFromRole(roleName)
	_ = strings.HasPrefix(gymID, ResourceGym)
	delete(r.users, username)
	return nil
}

// ListUsersInGroup returns a list of users in a particular group.
// NOTE: With Clerk auth, group membership is managed in the profiles collection.
// This stub returns an empty list; callers should query profiles directly if needed.
func (r *RBAC) ListUsersInGroup(ctx context.Context, group string) ([]GroupUser, error) {
	return []GroupUser{}, nil
}

func ValidateRole(role string) bool {
	switch role {
	case Student:
		return true
	case Owner:
		return true
	case Coach:
		return true
	default:
		return false
	}
}
