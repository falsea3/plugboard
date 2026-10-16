package dbeaver

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
)

const credentialsKey = "babb4a9f774ab853c96c2d653dfe544a"

type secret struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type credentials map[string]secret

func readCredentials(path string) (map[string]credentials, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]credentials{}, nil
	}
	if err != nil {
		return nil, err
	}
	plain, err := decrypt(b)
	if err != nil {
		return nil, errors.New("DBeaver's saved passwords can't be read: " + err.Error())
	}
	out := map[string]credentials{}
	if err := json.Unmarshal(plain, &out); err != nil {
		return nil, errors.New("DBeaver's saved passwords can't be read: " + err.Error())
	}
	return out, nil
}

func decrypt(b []byte) ([]byte, error) {
	key, _ := hex.DecodeString(credentialsKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(b) < 2*aes.BlockSize || len(b)%aes.BlockSize != 0 {
		return nil, errors.New("the file has an unexpected size")
	}
	iv, data := b[:aes.BlockSize], b[aes.BlockSize:]
	out := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, data)
	pad := int(out[len(out)-1])
	if pad == 0 || pad > aes.BlockSize || pad > len(out) {
		return nil, errors.New("the file isn't encrypted the way DBeaver does it")
	}
	return out[:len(out)-pad], nil
}
