package rbac

import (
	"context"
	"fmt"
	"strings"

	"github.com/Grapple-2024/backend/internal/service/profiles"
	"github.com/Grapple-2024/backend/pkg/cognito"
	"github.com/Grapple-2024/backend/pkg/utils"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
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
// Users inherit Cognito API's types.UserType.
type User struct {
	types.UserType

	ID    string
	Roles []string
}

// UserStore interface for fetching user data.
type UserStore interface {
	GetUser(ctx context.Context, userID string) (*User, error)
}

// RBAC is the core RBAC object.
type RBAC struct {
	*cognito.Client

	users       map[string]User
	roles       map[string]Role
	permissions map[string]Permission
}

func New(profileSVC *profiles.Service, cognito *cognito.Client) (*RBAC, error) {
	r := &RBAC{
		roles:       make(map[string]Role),
		permissions: make(map[string]Permission),
		Client:      cognito,
	}

	if err := r.SeedCache(context.Background()); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *RBAC) GetUser(ctx context.Context, userID string) (*User, error) {
	if val, ok := r.users[userID]; ok {
		return &val, nil
	}

	// send API request only if user is not in cache
	resp, err := r.ListGroupsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	u := &User{
		ID: userID,
	}
	for _, g := range resp.Groups {
		u.Roles = append(u.Roles, *g.GroupName)
	}
	log.Info().Msgf("User %q has the following roles: %v", userID, u.Roles)

	return u, nil
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
	user, err := r.GetUser(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	permissionNeeded := fmt.Sprintf("%s:%s", resource, action)
	totalPermissions := []string{} // just used to output what permissions the user has for debugging
	for _, roleName := range user.Roles {
		role, ok := r.roles[roleName]
		if !ok {
			log.Warn().Msgf("could not find role '%s' in role cache: %v", roleName, r.roles)
			continue
		}
		totalPermissions = append(totalPermissions, role.Permissions...)

		for _, userPermission := range role.Permissions {
			if userPermission == permissionNeeded {
				return true, nil
			}
		}
	}

	log.Warn().Msgf("User does not have permission for %s, user's permissions are: %v", permissionNeeded, totalPermissions)

	return false, nil
}

// CreateGymGroups creates Cognito groups and stores roles and permissions in RBAC cache for a new gym.
// This function is called when a new gym is created and is part of the gym creation transaction.
func (r *RBAC) CreateGymRBAC(ctx context.Context, gymID string) error {
	var groups = []string{"owners", "coaches", "students"}

	for _, groupType := range groups {
		groupName := fmt.Sprintf("%s::%s::%s", ResourceGym, gymID, groupType)
		err := r.CreateGroup(ctx, groupName)
		if err != nil {
			return fmt.Errorf("failed to create group %s: %w", groupName, err)
		}
	}

	if err := r.StoreGymRBAC(gymID); err != nil {
		return err
	}
	return nil
}

// AssignUserToGymRole assigns a user to a specific gym's group (owner, coach, student, etc).
func (r *RBAC) AssignUserToGymRole(ctx context.Context, gymID, username, roleName string) error {
	user, err := r.GetUser(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to find user in RBAC system: %v", err)
	}

	groupName := fmt.Sprintf("%s::%s::%s", ResourceGym, gymID, utils.PluralGroupNameFromRole(roleName))
	gymGroupPrefix := strings.Join(strings.Split(groupName, "::")[:2], "::")
	for _, role := range user.Roles {
		if !strings.HasPrefix(role, gymGroupPrefix) {
			continue
		}
		if err := r.RemoveUserFromGroup(ctx, username, role); err != nil {
			return err
		}
	}

	if err := r.AddUserToGroup(ctx, username, groupName); err != nil {
		return fmt.Errorf("failed to add user %s to group %s: %w", username, groupName, err)
	}

	// invalidate the cache for this user so we force the next RBAC IsAuthorized check to pull from cognito
	delete(r.users, username)
	return nil
}

// ListUsersInGroupByGym returns a list of users that are in a particular group.
func (r *RBAC) ListUsersInGroup(ctx context.Context, group string) ([]types.UserType, error) {
	paginator := cip.NewListUsersInGroupPaginator(r.Client, &cip.ListUsersInGroupInput{
		GroupName: &group,
	})

	var users []types.UserType
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list users in group %q: %w", group, err)
		}
		users = append(users, page.Users...)
	}

	return users, nil
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
