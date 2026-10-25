package account

// gRPC trailers on the account calls, for what their responses have no field
// for (the agent sets them, the CLI reads them).
const (
	// TrailerOldRoleName: update-role switched a legacy account to a role
	// from the CloudFormation template; this is the old role, which the user
	// deletes.
	TrailerOldRoleName = "pennsieve-old-role-name"
	// TrailerRoleStack: deregister removed an account whose role is a
	// CloudFormation stack, named so, which the user may delete.
	TrailerRoleStack = "pennsieve-role-stack"
)
