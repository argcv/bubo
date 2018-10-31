package msg

import "crypto/sha1"

type AuthRequest struct {
	Port  int
	Token string
}

func (nr *AuthRequest) CheckSecret(secret string) bool {
	return HashSecret(secret) == nr.Token
}

func NewAuthRequest(port int, secret string) *AuthRequest {
	return &AuthRequest{
		Port:  port,
		Token: HashSecret(secret),
	}
}

func HashSecret(secret string) string {
	return string(sha1.New().Sum([]byte(secret)))
}

type AuthAck struct {
	Success bool
	Message string
}

