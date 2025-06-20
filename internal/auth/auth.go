package auth

type Authenticator interface {
	GenerateToken() (string, error)
	ValidateToken() error
}
