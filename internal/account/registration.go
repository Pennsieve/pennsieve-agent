package account

type Registration interface {
	Create() ([]byte, error)
}
