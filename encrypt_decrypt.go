package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	KeyEnv    = "PRIVATE_KEY_B64"
	NonceSize = 12
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: gen-key | encrypt <infile> <outfile> | decrypt <infile> <outfile>")
		return
	}
	cmd := os.Args[1]
	switch cmd {
	case "gen-key":
		genKey()
	case "encrypt":
		if len(os.Args) != 4 { fmt.Println("encrypt <in> <out>"); return }
		if err := encryptFile(os.Args[2], os.Args[3]); err != nil { fmt.Fprintln(os.Stderr, "encrypt:", err) }
	case "decrypt":
		if len(os.Args) != 4 { fmt.Println("decrypt <in> <out>"); return }
		if err := decryptFile(os.Args[2], os.Args[3]); err != nil { fmt.Fprintln(os.Stderr, "decrypt:", err) }
	default:
		fmt.Println("unknown command")
	}
}

func genKey() {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil { fmt.Fprintln(os.Stderr, "gen-key:", err); return }
	fmt.Println(base64.StdEncoding.EncodeToString(key))
}

func getKey() ([]byte, error) {
	k := os.Getenv(KeyEnv)
	if k == "" { return nil, fmt.Errorf("%s not set", KeyEnv) }
	key, err := base64.StdEncoding.DecodeString(k)
	if err != nil { return nil, err }
	if len(key) != 32 { return nil, fmt.Errorf("invalid key length %d", len(key)) }
	return key, nil
}

func encryptFile(in, out string) error {
	key, err := getKey()
	if err != nil { return err }
	plain, err := os.ReadFile(in)
	if err != nil { return err }
	block, err := aes.NewCipher(key); if err != nil { return err }
	gcm, err := cipher.NewGCMWithNonceSize(block, NonceSize); if err != nil { return err }

	nonce := make([]byte, NonceSize); if _, err := io.ReadFull(rand.Reader, nonce); err != nil { return err }
	ct := gcm.Seal(nil, nonce, plain, nil)
	outb := append(nonce, ct...)
	return os.WriteFile(out, []byte(base64.StdEncoding.EncodeToString(outb)+"\n"), 0600)
}

func decryptFile(in, out string) error {
	key, err := getKey()
	if err != nil { return err }
	b64, err := os.ReadFile(in); if err != nil { return err }
	data, err := base64.StdEncoding.DecodeString(string(b64)); if err != nil { return err }
	if len(data) < NonceSize { return errors.New("ciphertext too short") }

	nonce := data[:NonceSize]; ct := data[NonceSize:]
	block, err := aes.NewCipher(key); if err != nil { return err }
	gcm, err := cipher.NewGCMWithNonceSize(block, NonceSize); if err != nil { return err }
	plain, err := gcm.Open(nil, nonce, ct, nil); if err != nil { return err }
	return os.WriteFile(out, plain, 0600)
}
