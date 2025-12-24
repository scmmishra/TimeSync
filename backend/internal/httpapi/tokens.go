package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"math/big"
)

const (
	tokenBytes = 32
	codeLength = 8
)

var codeAlphabet = []rune("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")

func generateToken() (string, []byte, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	hash := sha256.Sum256([]byte(token))
	return token, hash[:], nil
}

func generateCode() (string, error) {
	out := make([]rune, codeLength)
	for i := 0; i < codeLength; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(codeAlphabet))))
		if err != nil {
			return "", err
		}
		out[i] = codeAlphabet[n.Int64()]
	}
	return string(out), nil
}

func hashString(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}
