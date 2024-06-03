//go:build ignore

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
)

func main() {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		panic(err)
	}

	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	}

	file, err := os.Create("disgord.pem")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	if err := pem.Encode(file, block); err != nil {
		panic(err)
	}
}
