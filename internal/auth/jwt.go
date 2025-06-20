package auth

type JWTAuthenticator struct {
	secret string
	aud    string
	iss    string
}

func (a *JWTAuthenticator) NewJWTAuthenticator(secret, aud, iss string) *JWTAuthenticator {
	return &JWTAuthenticator{
		secret: secret,
		aud:    aud,
		iss:    iss,
	}
}

func (a *JWTAuthenticator) GenerateToken() (string, error) {

	return "", nil
}
func (a *JWTAuthenticator) ValidateToken() error {
	return nil
}
