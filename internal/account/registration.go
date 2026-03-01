package account

type Registration interface {
	Create() ([]byte, error)
	Delete() error
	GetAccountId() (string, error)
}
