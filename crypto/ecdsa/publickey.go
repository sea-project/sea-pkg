package ecdsa

import (
	e "crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"github.com/sea-project/sea-pkg/crypto/sha3"
)

const (
	// 序列化公钥的长度
	PubKeyBytesLenUncompressed = 65

	// x coord + y coord
	pubkeyUncompressed byte = 0x4
)

type PublicKey e.PublicKey

// ToECDSA 将公钥作为*ecdsa.PublicKey返回
func (p *PublicKey) ToECDSA() *e.PublicKey {
	return (*e.PublicKey)(p)
}

// IsEqual 公钥相等判断
func (p *PublicKey) IsEqual(otherPubKey *PublicKey) bool {
	return p.X.Cmp(otherPubKey.X) == 0 && p.Y.Cmp(otherPubKey.Y) == 0
}

// PubKeyToHex 公钥转哈希字符串
func (p *PublicKey) ToHex() string {

	// 将公钥序列化为65位非压缩
	unCompress := p.SerializeUncompressed()

	// 将Byte类型65位非压缩转哈希字符串
	return hex.EncodeToString(unCompress)
}

// SerializeUncompressed 将公钥序列化为65位的[]byte
func (p *PublicKey) SerializeUncompressed() []byte {
	b := make([]byte, 0, PubKeyBytesLenUncompressed)
	b = append(b, pubkeyUncompressed)
	b = paddedAppend(32, b, p.X.Bytes())
	return paddedAppend(32, b, p.Y.Bytes())
}

// PubkeyToAddress 公钥转地址方法
func (p *PublicKey) ToAddress() Address {
	pubBytes := p.FromECDSAPub()
	i := sha3.Keccak256(pubBytes[1:])[12:]
	return BytesToAddress(i)
}

// FromECDSAPub 椭圆加密公钥转坐标
func (p *PublicKey) FromECDSAPub() []byte {
	if p == nil || p.X == nil || p.Y == nil {
		return nil
	}
	return elliptic.Marshal(S256(), p.X, p.Y)
}

// HexToPubKey 哈希字符串转换secp256k1公钥
func HexToPubKey(hexkey string) (*PublicKey, error) {
	b, err := hex.DecodeString(hexkey)
	if err != nil {
		return nil, errors.New("invalid hex string")
	}
	return UnmarshalPubkey(b)
}

// UnmarshalPubkey 将[]byte转换为secp256k1公钥
func UnmarshalPubkey(pub []byte) (*PublicKey, error) {
	x, y := elliptic.Unmarshal(S256(), pub)
	if x == nil {
		return nil, errors.New("invalid secp256k1 public key")
	}
	return &PublicKey{Curve: S256(), X: x, Y: y}, nil
}

// paddedAppend 将src字节片追加到dst，返回新的片。如果源的长度小于传递的大小，则在添加src之前先将前置零字节附加到dst片。
func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}
