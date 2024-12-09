package account

type Registration interface {
	Create() ([]byte, error)
	GetAccountId() (string, error)
}
