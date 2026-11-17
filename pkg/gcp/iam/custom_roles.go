package iam

import (
	"context"

	"google.golang.org/api/iam/v1"
)

func (c *Client) GetServiceAccount(ctx context.Context, name string) (*iam.ServiceAccount, error) {
	return c.iamService.Projects.ServiceAccounts.Get(name).Context(ctx).Do()
}
func (c *Client) GetCustomRole(ctx context.Context, name string) (*iam.Role, error) {
	return c.iamService.Projects.Roles.Get(name).Context(ctx).Do()
}
func (c *Client) CreateCustomRole(ctx context.Context, id string, role *iam.Role) (*iam.Role, error) {
	return c.iamService.Projects.Roles.Create("projects/"+c.ProjectID, &iam.CreateRoleRequest{RoleId: id, Role: role}).Context(ctx).Do()
}
func (c *Client) UpdateCustomRole(ctx context.Context, name string, role *iam.Role) (*iam.Role, error) {
	return c.iamService.Projects.Roles.Patch(name, role).UpdateMask("includedPermissions,description,title,stage").Context(ctx).Do()
}
func (c *Client) DeleteCustomRole(ctx context.Context, name, etag string) error {
	_, err := c.iamService.Projects.Roles.Delete(name).Etag(etag).Context(ctx).Do()
	return err
}

func (c *Client) StartPool(ctx context.Context, parent, id string, pool *iam.WorkloadIdentityPool) (string, error) {
	op, err := c.iamService.Projects.Locations.WorkloadIdentityPools.Create(parent, pool).WorkloadIdentityPoolId(id).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return op.Name, nil
}
func (c *Client) StartProvider(ctx context.Context, parent, id string, provider *iam.WorkloadIdentityPoolProvider) (string, error) {
	op, err := c.iamService.Projects.Locations.WorkloadIdentityPools.Providers.Create(parent, provider).WorkloadIdentityPoolProviderId(id).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return op.Name, nil
}
func (c *Client) WaitOperation(ctx context.Context, name string) error {
	return c.waitOperation(ctx, name)
}

// CheckCustomPermissions is read-only; unsupported/unknown permissions are not
// silently dropped or substituted with predefined roles.
func (c *Client) CheckCustomPermissions(ctx context.Context, permissions []string) error {
	wanted := map[string]bool{}
	for _, p := range permissions {
		wanted[p] = false
	}
	err := c.iamService.Permissions.QueryTestablePermissions(&iam.QueryTestablePermissionsRequest{FullResourceName: "//cloudresourcemanager.googleapis.com/projects/" + c.ProjectID}).Pages(ctx, func(page *iam.QueryTestablePermissionsResponse) error {
		for _, p := range page.Permissions {
			if _, ok := wanted[p.Name]; ok && (p.CustomRolesSupportLevel == "SUPPORTED" || p.CustomRolesSupportLevel == "TESTING") && !p.OnlyInPredefinedRoles {
				wanted[p.Name] = true
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, ok := range wanted {
		if !ok {
			return errUnsupportedPermissions{}
		}
	}
	return nil
}

type errUnsupportedPermissions struct{}

func (errUnsupportedPermissions) SafeReason() string {
	return "contract has unsupported or unavailable project custom-role permissions"
}

func (errUnsupportedPermissions) Error() string {
	return "contract has unsupported or unavailable project custom-role permissions"
}
