package aws

type AWSRole struct {
	Role RoleDetail
}

type RoleDetail struct {
	RoleName string
	RoleId   string
	Arn      string
}
