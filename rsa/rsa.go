package rsa

import (
	"log"
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

type Keys struct {
	signingKey, verificationKey []byte
}

func (k Keys) PrivateKey() []byte{
	return k.signingKey
}

func (k Keys) PublicKey() []byte {
	return k.verificationKey
}

func (k *Keys) InitKeys() {
	var (
		err error
		privKey *rsa.PrivateKey
		pubKey *rsa.PublicKey
		pubKeyBytes []byte
	)
	privKey, err = rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		log.Fatal("Error generating private key")
	}
	pubKey = &privKey.PublicKey

	var privPEMBlock = &pem.Block{
		Type: "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	}
	privKeyPEMBuffer := new(bytes.Buffer)
	pem.Encode(privKeyPEMBuffer, privPEMBlock)
	k.signingKey = privKeyPEMBuffer.Bytes()

	pubKeyBytes, err = x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		log.Fatal("Error marshalling public key")
	}

	var pubPEMBlock = &pem.Block{
		Type: "RSA PUBLIC KEY",
		Bytes: pubKeyBytes,
	}
	pubKeyPEMBuffer := new(bytes.Buffer)
	pem.Encode(pubKeyPEMBuffer, pubPEMBlock)
	k.verificationKey = pubKeyPEMBuffer.Bytes()
}
