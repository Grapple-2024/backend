package rbac

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"path"
	"text/template"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type RBACConfig struct {
	Roles       *RolesConfig
	Permissions *PermissionsConfig
}

type RolesConfig struct {
	Static       []Role `json:"static"`
	DynamicGymID []Role `json:"dynamic_gym_id"`
}

type PermissionsConfig struct {
	Static       []Permission `json:"static"`
	DynamicGymID []Permission `json:"dynamic_gym_id"`
}

//go:embed static/*.json
var rolePermissionConfig embed.FS

func renderTemplate(gymID string, templateFile string) ([]byte, error) {
	tplName := path.Base(templateFile)

	tpl, err := template.New(tplName).ParseFS(rolePermissionConfig, templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse FS: %v", err)
	}

	data := struct {
		GymID string
	}{
		GymID: gymID,
	}

	var result bytes.Buffer
	if err := tpl.Execute(&result, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %v", err)
	}

	return result.Bytes(), nil
}

func (r *RBAC) RenderPermissionsTemplate(gymID string) (*PermissionsConfig, error) {
	result, err := renderTemplate(gymID, "static/permissions.json")
	if err != nil {
		return nil, err
	}

	var permissionsConfig PermissionsConfig
	if err := json.Unmarshal(result, &permissionsConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permissions config: %v", err)
	}

	return &permissionsConfig, nil
}

func (r *RBAC) RenderRolesTemplate(gymID string) (*RolesConfig, error) {
	result, err := renderTemplate(gymID, "static/roles.json")
	if err != nil {
		return nil, fmt.Errorf("failed to render roles template: %v", err)
	}

	var roleConfig RolesConfig
	if err := json.Unmarshal(result, &roleConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal roles config: %v", err)
	}

	return &roleConfig, nil
}

func GetRBACConfig() (*RBACConfig, error) {
	permissionConfigBytes, err := rolePermissionConfig.ReadFile("static/permissions.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read permissions.json: %v", err)
	}

	roleConfigBytes, err := rolePermissionConfig.ReadFile("static/roles.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read roles.json: %v", err)
	}

	rolesConfig := RolesConfig{}
	if err := json.Unmarshal(roleConfigBytes, &rolesConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal roles config: %v", err)
	}

	permissionsConfig := PermissionsConfig{}
	if err := json.Unmarshal(permissionConfigBytes, &permissionsConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permissions config: %v", err)
	}

	return &RBACConfig{
		Roles:       &rolesConfig,
		Permissions: &permissionsConfig,
	}, nil
}

func (r *RBAC) StoreGymRBAC(gymID string) error {
	rolesConfig, err := r.RenderRolesTemplate(gymID)
	if err != nil {
		return fmt.Errorf("failed to render roles template: %v", err)
	}
	permissionsConfig, err := r.RenderPermissionsTemplate(gymID)
	if err != nil {
		return err
	}
	r.AddPermissions(permissionsConfig.DynamicGymID...)
	r.AddRoles(rolesConfig.DynamicGymID...)

	return nil
}

// SeedCache loads static permissions/roles and all dynamic gym roles from MongoDB.
// It runs once: subsequent calls are no-ops (guarded by r.seeded).
func (r *RBAC) SeedCache(ctx context.Context) error {
	if r.seeded {
		return nil
	}

	rbacConfig, err := GetRBACConfig()
	if err != nil {
		return err
	}
	r.AddRoles(rbacConfig.Roles.Static...)
	r.AddPermissions(rbacConfig.Permissions.Static...)

	// Load dynamic roles for every gym that already exists in MongoDB.
	gymIDs, err := r.allGymIDs(ctx)
	if err != nil {
		return fmt.Errorf("SeedCache: failed to list gym IDs: %w", err)
	}
	for _, gymID := range gymIDs {
		if err := r.StoreGymRBAC(gymID); err != nil {
			log.Warn().Msgf("SeedCache: could not load RBAC for gym %q: %v", gymID, err)
		}
	}

	r.seeded = true
	log.Info().Msgf("RBAC cache seeded: %d static roles, %d static permissions, %d gyms",
		len(rbacConfig.Roles.Static), len(rbacConfig.Permissions.Static), len(gymIDs))
	return nil
}

// allGymIDs returns the hex IDs of every gym in MongoDB.
func (r *RBAC) allGymIDs(ctx context.Context) ([]string, error) {
	gyms := r.mongoClient.Database("grapple").Collection("gyms")

	cursor, err := gyms.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		var doc struct {
			ID bson.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			log.Warn().Msgf("allGymIDs: failed to decode: %v", err)
			continue
		}
		ids = append(ids, doc.ID.Hex())
	}
	return ids, cursor.Err()
}
