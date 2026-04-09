package rbac

import (
	"context"
	"fmt"

	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"
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

// RBAC is the core RBAC object.
type RBAC struct {
	mongoClient *mongoext.Client
	roles       map[string]Role
	permissions map[string]Permission
	seeded      bool
}

func New(mongoClient *mongoext.Client) (*RBAC, error) {
	r := &RBAC{
		mongoClient: mongoClient,
		roles:       make(map[string]Role),
		permissions: make(map[string]Permission),
	}
	return r, nil
}

// GetUser fetches the user's gym group memberships from MongoDB and returns them as RBAC roles.
func (r *RBAC) GetUser(ctx context.Context, userID string) (*User, error) {
	profiles := r.mongoClient.Database("grapple").Collection("profiles")
	filter := bson.M{"cognito_id": userID}

	var profile struct {
		Gyms []struct {
			Group string `bson:"group"`
		} `bson:"gyms"`
	}
	if err := profiles.FindOne(ctx, filter).Decode(&profile); err != nil {
		return nil, fmt.Errorf("profile not found for user %q: %w", userID, err)
	}

	roles := make([]string, 0, len(profile.Gyms))
	for _, g := range profile.Gyms {
		if g.Group != "" {
			roles = append(roles, g.Group)
		}
	}

	log.Info().Msgf("GetUser(%q): resolved roles %v", userID, roles)
	return &User{ID: userID, Roles: roles}, nil
}

// AddRoles adds one or more roles to the RBAC system (in-memory cache).
func (r *RBAC) AddRoles(roles ...Role) {
	for _, role := range roles {
		r.roles[role.Name] = role
	}
}

// AddPermissions adds new permissions to the RBAC system (in-memory cache).
func (r *RBAC) AddPermissions(permissions ...Permission) {
	for _, p := range permissions {
		permission := fmt.Sprintf("%s:%s", p.Resource, p.Action)
		if _, ok := r.permissions[permission]; !ok {
			r.permissions[permission] = p
		}
	}
}

// IsAuthorized checks if a user is authorized to perform an action on a resource.
// It seeds the role/permission cache once on first call, then looks up the user's
// gym group memberships from MongoDB and checks them against the in-memory permission map.
func (r *RBAC) IsAuthorized(ctx context.Context, userID, resource, action string) (bool, error) {
	if !r.seeded {
		if err := r.SeedCache(ctx); err != nil {
			return false, fmt.Errorf("failed to seed RBAC cache: %w", err)
		}
	}

	user, err := r.GetUser(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	permissionNeeded := fmt.Sprintf("%s:%s", resource, action)
	log.Info().Msgf("IsAuthorized(%q, %q)?", userID, permissionNeeded)

	for _, roleName := range user.Roles {
		// Ensure this gym's dynamic roles are loaded (idempotent).
		gymID := gymIDFromGroup(roleName)
		if gymID != "" {
			if err := r.StoreGymRBAC(gymID); err != nil {
				log.Warn().Msgf("could not load RBAC for gym %q: %v", gymID, err)
			}
		}

		role, ok := r.roles[roleName]
		if !ok {
			log.Warn().Msgf("role %q not in cache after StoreGymRBAC", roleName)
			continue
		}

		for _, p := range role.Permissions {
			if p == permissionNeeded {
				return true, nil
			}
		}
	}

	log.Warn().Msgf("user %q denied %q: roles=%v", userID, permissionNeeded, user.Roles)
	return false, nil
}

// CreateGymRBAC loads roles and permissions for a new gym into the in-memory cache.
func (r *RBAC) CreateGymRBAC(ctx context.Context, gymID string) error {
	return r.StoreGymRBAC(gymID)
}

// AssignUserToGymRole ensures the gym's role config is loaded into cache.
// The actual role assignment is persisted by UpsertGymAssociation in the profiles service.
func (r *RBAC) AssignUserToGymRole(ctx context.Context, gymID, username, roleName string) error {
	return r.StoreGymRBAC(gymID)
}

// RemoveUserFromGymGroups is a no-op — membership is managed via Profile.gyms in MongoDB.
func (r *RBAC) RemoveUserFromGymGroups(ctx context.Context, gymID, userID string) error {
	return nil
}

// ListUsersInGroup queries MongoDB for all profiles whose gyms array contains the given group name,
// and returns their emails. Used to send notifications to coaches/owners.
func (r *RBAC) ListUsersInGroup(ctx context.Context, group string) ([]GroupUser, error) {
	profiles := r.mongoClient.Database("grapple").Collection("profiles")
	filter := bson.M{
		"gyms": bson.M{
			"$elemMatch": bson.M{"group": group},
		},
	}

	cursor, err := profiles.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("ListUsersInGroup(%q): query failed: %w", group, err)
	}
	defer cursor.Close(ctx)

	var results []GroupUser
	for cursor.Next(ctx) {
		var profile struct {
			Email string `bson:"email"`
		}
		if err := cursor.Decode(&profile); err != nil {
			log.Warn().Msgf("ListUsersInGroup: failed to decode profile: %v", err)
			continue
		}
		if profile.Email != "" {
			results = append(results, GroupUser{Username: profile.Email})
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("ListUsersInGroup(%q): cursor error: %w", group, err)
	}

	log.Info().Msgf("ListUsersInGroup(%q): found %d users", group, len(results))
	return results, nil
}

func ValidateRole(role string) bool {
	switch role {
	case Student, Owner, Coach:
		return true
	default:
		return false
	}
}

// gymIDFromGroup extracts the gym ID from a group name like "gym::{gymID}::owners".
// Returns "" if the group name is not a dynamic gym group.
func gymIDFromGroup(group string) string {
	// Expected format: gym::<gymID>::<role>
	var prefix, gymID, _ string
	n, _ := fmt.Sscanf(group, "%s", &prefix) // just check it's non-empty
	_ = n
	parts := splitGroup(group)
	if len(parts) == 3 && parts[0] == "gym" {
		gymID = parts[1]
	}
	return gymID
}

// splitGroup splits "gym::{id}::owners" on "::" into ["gym", "{id}", "owners"].
func splitGroup(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s)-1; i++ {
		if s[i] == ':' && s[i+1] == ':' {
			parts = append(parts, s[start:i])
			start = i + 2
			i++ // skip second ':'
		}
	}
	parts = append(parts, s[start:])
	return parts
}
