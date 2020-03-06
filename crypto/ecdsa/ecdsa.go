package ecdsa

import (
	"bytes"
	e "crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
)

// GenerateKey 随机生成公私钥对
func GenerateKey() (*PrivateKey, *PublicKey) {
	randm := rand.Reader
	randBytes := make([]byte, 64)
	randm.Read(randBytes)
	reader := bytes.NewReader(randBytes)
	prvKey, _ := e.GenerateKey(S256(), reader)
	return PrivKeyFromBytes(S256(), prvKey.D.Bytes())
}

// PrivKeyFromBytes 根据私钥随机数D返回公私钥
func PrivKeyFromBytes(curve elliptic.Curve, pk []byte) (*PrivateKey, *PublicKey) {
	x, y := curve.ScalarBaseMult(pk)
	priv := &PrivateKey{
		PublicKey: e.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(pk),
	}
	return priv, (*PublicKey)(&priv.PublicKey)
}

// GenerateSharedSecret 基于私钥和公钥生成共享密钥。
func GenerateSharedSecret(privkey *PrivateKey, pubkey *PublicKey) []byte {
	x, _ := pubkey.Curve.ScalarMult(pubkey.X, pubkey.Y, privkey.D.Bytes())
	return x.Bytes()
}
