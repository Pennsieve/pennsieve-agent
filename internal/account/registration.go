package account

type Registration interface {
	Create() ([]byte, error)
	Update() error
	Delete() error
	GetAccountId() (string, error)
}
