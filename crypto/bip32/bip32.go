package bip32

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sea-project/sea-pkg/crypto/base58"
	"github.com/sea-project/sea-pkg/crypto/ecdsa"
	"golang.org/x/crypto/ripemd160"
	"io"
	"math/big"
)

const (
	FirstHardenedChild        = uint32(0x80000000)
	PublicKeyCompressedLength = 33
	BitcoinSeed               = "Bitcoin seed"
)

var (
	// 私钥增加xpr前缀
	PrivateWalletVersion, _ = hex.DecodeString("0488ADE4")
	// 私钥增加xpr前缀
	PublicWalletVersion, _    = hex.DecodeString("0488B21E")
	ErrSerializedKeyWrongSize = errors.New("Serialized keys should by exactly 82 bytes")
	ErrHardnedChildPublicKey  = errors.New("Can't create hardened child for public key")
	ErrInvalidChecksum        = errors.New("Checksum doesn't match")
	ErrInvalidPrivateKey      = errors.New("Invalid private key")
	ErrInvalidPublicKey       = errors.New("Invalid public key")

	curve                 = ecdsa.S256()
	curveParams           = curve.Params()
	BitcoinBase58Encoding = base58.NewAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
)

// Key represents a bip32 extended key
type Key struct {
	Key         []byte `json:"key"`          // 33 bytes
	Version     []byte `json:"version"`      // 4 bytes
	ChildNumber []byte `json:"child_number"` // 4 bytes bip44层级
	FingerPrint []byte `json:"finger_print"` // 4 bytes
	ChainCode   []byte `json:"chain_code"`   // 32 bytes
	Depth       byte   `json:"depth"`        // 1 bytes
	IsPrivate   bool   `json:"is_private"`   // unserialized
}

// NewMasterKey creates a new master extended key from a seed
func NewMasterKey(seed []byte) (*Key, error) {
	// Generate key and chaincode
	hmac := hmac.New(sha512.New, []byte(BitcoinSeed))
	_, err := hmac.Write(seed)
	if err != nil {
		return nil, err
	}
	intermediary := hmac.Sum(nil)

	// Split it into our key and chain code
	keyBytes := intermediary[:32]  // 私钥
	chainCode := intermediary[32:] // chainCode

	return NewMasterKey2(keyBytes, chainCode)
}

func NewMasterKey2(keyBytes, chainCode []byte) (*Key, error) {
	// Validate key
	err := validatePrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}

	// Create the key struct
	key := &Key{
		Version:     PrivateWalletVersion,
		ChainCode:   chainCode,
		Key:         keyBytes,
		Depth:       0x0,
		ChildNumber: []byte{0x00, 0x00, 0x00, 0x00},
		FingerPrint: []byte{0x00, 0x00, 0x00, 0x00},
		IsPrivate:   true,
	}

	return key, nil
}

// NewChildKey derives a child key from a given parent as outlined by bip32
func (key *Key) NewChildKey(childIdx uint32) (*Key, error) {
	// Fail early if trying to create hardned child from public key
	if !key.IsPrivate && childIdx >= FirstHardenedChild {
		return nil, ErrHardnedChildPublicKey
	}

	intermediary, err := key.getIntermediary(childIdx)
	if err != nil {
		return nil, err
	}

	// Create child Key with data common to all both scenarios
	childKey := &Key{
		ChildNumber: uint32Bytes(childIdx),
		ChainCode:   intermediary[32:],
		Depth:       key.Depth + 1,
		IsPrivate:   key.IsPrivate,
	}

	// Bip32 CKDpriv
	if key.IsPrivate {
		childKey.Version = PrivateWalletVersion
		fingerprint, err := hash160(publicKeyForPrivateKey(key.Key))
		if err != nil {
			return nil, err
		}
		childKey.FingerPrint = fingerprint[:4]
		childKey.Key = addPrivateKeys(intermediary[:32], key.Key)

		// Validate key
		err = validatePrivateKey(childKey.Key)
		if err != nil {
			return nil, err
		}
		// Bip32 CKDpub
	} else {
		keyBytes := publicKeyForPrivateKey(intermediary[:32])

		// Validate key
		err := validateChildPublicKey(keyBytes)
		if err != nil {
			return nil, err
		}

		childKey.Version = PublicWalletVersion
		fingerprint, err := hash160(key.Key)
		if err != nil {
			return nil, err
		}
		childKey.FingerPrint = fingerprint[:4]
		childKey.Key = addPublicKeys(keyBytes, key.Key)
	}

	return childKey, nil
}

func (key *Key) getIntermediary(childIdx uint32) ([]byte, error) {
	// Get intermediary to create key and chaincode from
	// Hardened children are based on the private key
	// NonHardened children are based on the public key
	childIndexBytes := uint32Bytes(childIdx)

	var data []byte
	if childIdx >= FirstHardenedChild {
		data = append([]byte{0x0}, key.Key...)
	} else {
		if key.IsPrivate {
			data = publicKeyForPrivateKey(key.Key)
		} else {
			data = key.Key
		}
	}
	data = append(data, childIndexBytes...)

	hmac := hmac.New(sha512.New, key.ChainCode)
	_, err := hmac.Write(data)
	if err != nil {
		return nil, err
	}
	return hmac.Sum(nil), nil
}

// PublicKey returns the public version of key or return a copy
// The 'Neuter' function from the bip32 spec
func (key *Key) PublicKey() *Key {
	keyBytes := key.Key

	if key.IsPrivate {
		keyBytes = publicKeyForPrivateKey(keyBytes)
	}

	return &Key{
		Version:     PublicWalletVersion,
		Key:         keyBytes,
		Depth:       key.Depth,
		ChildNumber: key.ChildNumber,
		FingerPrint: key.FingerPrint,
		ChainCode:   key.ChainCode,
		IsPrivate:   false,
	}
}

// Serialize a Key to a 78 byte byte slice
func (key *Key) Serialize() ([]byte, error) {
	// Private keys should be prepended with a single null byte
	keyBytes := key.Key
	if key.IsPrivate {
		keyBytes = append([]byte{0x0}, keyBytes...)
	}

	// Write fields to buffer in order
	buffer := new(bytes.Buffer)
	buffer.Write(key.Version)
	buffer.WriteByte(key.Depth)
	buffer.Write(key.FingerPrint)
	buffer.Write(key.ChildNumber)
	buffer.Write(key.ChainCode)
	buffer.Write(keyBytes)

	// Append the standard doublesha256 checksum
	serializedKey, err := addChecksumToBytes(buffer.Bytes())
	if err != nil {
		return nil, err
	}

	return serializedKey, nil
}

// B58Serialize encodes the Key in the standard Bitcoin base58 encoding
func (key *Key) B58Serialize() string {
	serializedKey, err := key.Serialize()
	if err != nil {
		return ""
	}

	return base58.Encode(serializedKey, BitcoinBase58Encoding)
}

// String encodes the Key in the standard Bitcoin base58 encoding
func (key *Key) String() string {
	return key.B58Serialize()
}

// Deserialize a byte slice into a Key
func Deserialize(data []byte) (*Key, error) {
	if len(data) != 82 {
		return nil, ErrSerializedKeyWrongSize
	}
	var key = &Key{}
	key.Version = data[0:4]
	key.Depth = data[4]
	key.FingerPrint = data[5:9]
	key.ChildNumber = data[9:13]
	key.ChainCode = data[13:45]

	if data[45] == byte(0) {
		key.IsPrivate = true
		key.Key = data[46:78]
	} else {
		key.IsPrivate = false
		key.Key = data[45:78]
	}

	// validate checksum
	cs1, err := checksum(data[0 : len(data)-4])
	if err != nil {
		return nil, err
	}

	cs2 := data[len(data)-4:]
	for i := range cs1 {
		if cs1[i] != cs2[i] {
			return nil, ErrInvalidChecksum
		}
	}
	return key, nil
}

// B58Deserialize deserializes a Key encoded in base58 encoding
func B58Deserialize(data string) (*Key, error) {
	b, err := base58.Decode(data, BitcoinBase58Encoding)
	if err != nil {
		return nil, err
	}
	return Deserialize(b)
}

// NewSeed returns a cryptographically secure seed
func NewSeed() ([]byte, error) {
	// Well that easy, just make go read 256 random bytes into a slice
	s := make([]byte, 256)
	_, err := rand.Read(s)
	return s, err
}

//
// Hashes
//

func hashSha256(data []byte) ([]byte, error) {
	hasher := sha256.New()
	_, err := hasher.Write(data)
	if err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}

func hashDoubleSha256(data []byte) ([]byte, error) {
	hash1, err := hashSha256(data)
	if err != nil {
		return nil, err
	}

	hash2, err := hashSha256(hash1)
	if err != nil {
		return nil, err
	}
	return hash2, nil
}

func hashRipeMD160(data []byte) ([]byte, error) {
	hasher := ripemd160.New()
	_, err := io.WriteString(hasher, string(data))
	if err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}

func hash160(data []byte) ([]byte, error) {
	hash1, err := hashSha256(data)
	if err != nil {
		return nil, err
	}

	hash2, err := hashRipeMD160(hash1)
	if err != nil {
		return nil, err
	}

	return hash2, nil
}

//
// Encoding
//

func checksum(data []byte) ([]byte, error) {
	hash, err := hashDoubleSha256(data)
	if err != nil {
		return nil, err
	}

	return hash[:4], nil
}

func addChecksumToBytes(data []byte) ([]byte, error) {
	checksum, err := checksum(data)
	if err != nil {
		return nil, err
	}
	return append(data, checksum...), nil
}

// Keys
func publicKeyForPrivateKey(key []byte) []byte {
	return compressPublicKey(curve.ScalarBaseMult(key))
}

func addPublicKeys(key1 []byte, key2 []byte) []byte {
	x1, y1 := expandPublicKey(key1)
	x2, y2 := expandPublicKey(key2)
	return compressPublicKey(curve.Add(x1, y1, x2, y2))
}

func addPrivateKeys(key1 []byte, key2 []byte) []byte {
	var key1Int big.Int
	var key2Int big.Int
	key1Int.SetBytes(key1)
	key2Int.SetBytes(key2)

	key1Int.Add(&key1Int, &key2Int)
	key1Int.Mod(&key1Int, curve.Params().N)

	b := key1Int.Bytes()
	if len(b) < 32 {
		extra := make([]byte, 32-len(b))
		b = append(extra, b...)
	}
	return b
}

func compressPublicKey(x *big.Int, y *big.Int) []byte {
	var key bytes.Buffer

	// Write header; 0x2 for even y value; 0x3 for odd
	key.WriteByte(byte(0x2) + byte(y.Bit(0)))

	// Write X coord; Pad the key so x is aligned with the LSB. Pad size is key length - header size (1) - xBytes size
	xBytes := x.Bytes()
	for i := 0; i < (PublicKeyCompressedLength - 1 - len(xBytes)); i++ {
		key.WriteByte(0x0)
	}
	key.Write(xBytes)

	return key.Bytes()
}

// As described at https://crypto.stackexchange.com/a/8916
func expandPublicKey(key []byte) (*big.Int, *big.Int) {
	Y := big.NewInt(0)
	X := big.NewInt(0)
	X.SetBytes(key[1:])

	// y^2 = x^3 + ax^2 + b
	// a = 0
	// => y^2 = x^3 + b
	ySquared := big.NewInt(0)
	ySquared.Exp(X, big.NewInt(3), nil)
	ySquared.Add(ySquared, curveParams.B)

	Y.ModSqrt(ySquared, curveParams.P)

	Ymod2 := big.NewInt(0)
	Ymod2.Mod(Y, big.NewInt(2))

	signY := uint64(key[0]) - 2
	if signY != Ymod2.Uint64() {
		Y.Sub(curveParams.P, Y)
	}

	return X, Y
}

func validatePrivateKey(key []byte) error {
	if fmt.Sprintf("%x", key) == "0000000000000000000000000000000000000000000000000000000000000000" || //if the key is zero
		bytes.Compare(key, curveParams.N.Bytes()) >= 0 || //or is outside of the curve
		len(key) != 32 { //or is too short
		return ErrInvalidPrivateKey
	}

	return nil
}

func validateChildPublicKey(key []byte) error {
	x, y := expandPublicKey(key)

	if x.Sign() == 0 || y.Sign() == 0 {
		return ErrInvalidPublicKey
	}

	return nil
}

//
// Numerical
//
func uint32Bytes(i uint32) []byte {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, i)
	return bytes
}

func PubKeyToAddr(pubKey []byte) string {
	hashPubKey, _ := hash160(pubKey)
	walletVersionedPayload := append([]byte{byte(0x00)}, hashPubKey...)
	checksum, _ := checksum(walletVersionedPayload)
	fullPayload := append(walletVersionedPayload, checksum...)
	address := base58.Encode(fullPayload, BitcoinBase58Encoding)

	return address
}

func JsonString(data interface{}) string {
	jsonByte, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
	}
	return string(jsonByte)
}

// 将HD显示的数字转换为uint32
// m/44'/60'/0'/0/account_index，如，那么传入44，将会转换为0x8000002C
func ParseHDNum(num uint32) uint32 {
	return num + FirstHardenedChild
}

// 将HD的uint32转换为可读数字
func FromHDNum(hdNum uint32) uint32 {
	return hdNum - FirstHardenedChild
}
