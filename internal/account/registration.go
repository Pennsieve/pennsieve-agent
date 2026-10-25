package account

type Registration interface {
	Delete() error
	GetAccountId() (string, error)
}
