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

// SeedCache loads the static permissions and roles caches for the RBAC framework to use during runtime.
// Dynamic gym roles are added via CreateGymRBAC when a gym is created.
// TODO: On startup, seed dynamic gym roles by querying gyms from MongoDB instead of Cognito.
func (r *RBAC) SeedCache(ctx context.Context) error {
	rbacConfig, err := GetRBACConfig()
	if err != nil {
		return err
	}

	r.AddRoles(rbacConfig.Roles.Static...)
	r.AddPermissions(rbacConfig.Permissions.Static...)

	log.Info().Msgf("Seeded %d static roles and %d static permissions", len(rbacConfig.Roles.Static), len(rbacConfig.Permissions.Static))

	return nil
}
