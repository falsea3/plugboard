package postgres

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strconv"

	"golang.org/x/text/secure/precis"
)

const scramIterations = 4096

func scramVerifier(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return scramWithSalt(password, salt, scramIterations)
}

func scramWithSalt(password string, salt []byte, iterations int) (string, error) {
	if prepared, err := precis.OpaqueString.String(password); err == nil {
		password = prepared
	}
	salted, err := pbkdf2.Key(sha256.New, password, salt, iterations, sha256.Size)
	if err != nil {
		return "", err
	}
	clientKey := mac(salted, "Client Key")
	storedKey := sha256.Sum256(clientKey)
	serverKey := mac(salted, "Server Key")
	b64 := base64.StdEncoding.EncodeToString
	return "SCRAM-SHA-256$" + strconv.Itoa(iterations) + ":" + b64(salt) + "$" + b64(storedKey[:]) + ":" + b64(serverKey), nil
}

func mac(key []byte, msg string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(msg))
	return h.Sum(nil)
}
