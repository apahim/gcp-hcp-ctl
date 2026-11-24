package platformapi

import (
	gcpv1 "github.com/openshift-online/gecko/platform-api/api/public/v1"
)

// The platform-scoped role catalog wire types come from the shared Platform API
// module, which registers them in the client scheme. They are aliased here so
// CLI callers keep using platformapi types. Nothing local is registered: adding
// a second type for the same group/version/kind makes the serializer panic.

// PlatformRole is the public, platform-scoped catalog wire type. Role
// definitions come from the server; the CLI does not supply permission bundles.
type PlatformRole = gcpv1.PlatformRole

// PlatformRoleSpec contains the complete permission set returned by the server.
type PlatformRoleSpec = gcpv1.PlatformRoleSpec

// PlatformRoleList is the Kubernetes-shaped public catalog response.
type PlatformRoleList = gcpv1.PlatformRoleList
