package rbac

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"path"
	"strings"
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

	log.Info().Msgf("Got rbac config: %+v", permissionsConfig)
	return &RBACConfig{
		Roles:       &rolesConfig,
		Permissions: &permissionsConfig,
	}, nil
}

func (r *RBAC) StoreGymRBAC(gymID string) error {
	rolesConfig, err := r.RenderRolesTemplate(gymID)
	if err != nil {
		return fmt.Errorf("faile to render roles template: %v", err)
	}
	permissionsConfig, err := r.RenderPermissionsTemplate(gymID)
	if err != nil {
		return err
	}
	r.AddPermissions(permissionsConfig.DynamicGymID...)
	r.AddRoles(rolesConfig.DynamicGymID...)

	return nil
}

// LoadCache loads the permissions and roles caches for the RBAC framework to use during runtime.
// It dynamically determines each gym by reading the Cognito groups API and populates the roles and permissions for each gym.
// It also populates any static roles/permissions, eg "gym-creator", and gym:create.
func (r *RBAC) SeedCache(ctx context.Context) error {
	rbacConfig, err := GetRBACConfig()
	if err != nil {
		return err
	}
	// add the static roles and permissions first
	r.AddRoles(rbacConfig.Roles.Static...)
	r.AddPermissions(rbacConfig.Permissions.Static...)

	// Add dynamic roles and permissions for each Gym found in Cognito
	resp, err := r.ListGroups(ctx)
	if err != nil {
		return err
	}

	gymRBACs := map[string]RBACConfig{}
	for _, g := range resp.Groups {
		parts := strings.Split(*g.GroupName, "::")
		if len(parts) < 3 {
			log.Debug().Msgf("Found non-dynamic group %s in Cognito! Skipping...", *g.GroupName)
			continue
		}
		gymID := parts[1]

		if _, ok := gymRBACs[gymID]; ok {
			continue
		}

		// render the configuration templates
		rolesConfig, err := r.RenderRolesTemplate(gymID)
		if err != nil {
			return err
		}
		permissionsConfig, err := r.RenderPermissionsTemplate(gymID)
		if err != nil {
			return err
		}

		gymRBACs[gymID] = RBACConfig{
			Roles:       rolesConfig,
			Permissions: permissionsConfig,
		}
	}

	for _, rbacConfig := range gymRBACs {
		r.AddPermissions(rbacConfig.Permissions.DynamicGymID...)
		r.AddRoles(rbacConfig.Roles.DynamicGymID...)
	}

	return nil
}
