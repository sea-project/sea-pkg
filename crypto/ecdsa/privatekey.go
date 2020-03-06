package ecdsa

import (
	e "crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/sea-project/sea-pkg/util/math"
	"math/big"
)

type PrivateKey e.PrivateKey

var (
	secp256k1N, _ = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
)

// ToPubKey 返回与此私钥对应的公钥
func (p *PrivateKey) ToPubKey() *PublicKey {
	return (*PublicKey)(&p.PublicKey)
}

// ToHex 私钥转哈希
func (p *PrivateKey) ToHex() string {
	return hex.EncodeToString(p.ToByte())
}

// ToByte 私钥转byte
func (p *PrivateKey) ToByte() []byte {
	if p == nil {
		return nil
	}
	return math.PaddedBigBytes(p.D, p.Params().BitSize/8)
}

// Sign 使用私钥为提供的散列(应该是散列较大消息的结果)生成ECDSA签名。生成的签名是确定性的(相同的消息和相同的密钥生成相同的签名)，并且符合RFC6979和BIP0062的规范。
func (p *PrivateKey) Sign(hash []byte) (*Signature, error) {
	return signRFC6979(p, hash)
}

// HexToECDSA 哈希字符串转私钥
func HexToPrvKey(hexkey string) (*PrivateKey, error) {
	b, err := hex.DecodeString(hexkey)
	if err != nil {
		return nil, errors.New("invalid hex string")
	}
	return ToECDSA(b, true)
}

// ToECDSA []byte转私钥
func ToECDSA(d []byte, strict bool) (*PrivateKey, error) {
	priv := new(PrivateKey)
	priv.PublicKey.Curve = S256()
	if strict && 8*len(d) != priv.Params().BitSize {
		return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
	}
	priv.D = new(big.Int).SetBytes(d)

	// The priv.D must < N
	if priv.D.Cmp(secp256k1N) >= 0 {
		return nil, fmt.Errorf("invalid private key, >=N")
	}
	// The priv.D must not be zero or negative.
	if priv.D.Sign() <= 0 {
		return nil, fmt.Errorf("invalid private key, zero or negative")
	}

	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	return priv, nil
}
