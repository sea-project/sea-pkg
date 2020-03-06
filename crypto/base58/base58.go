package base58

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// Alphabet: copy from https://en.wikipedia.org/wiki/Base58       自定义字母表
var (
	BscAlphabet     = NewAlphabet("1Aa2Bb3Cc4Dd5Ee6Ff7Gg8Hh9jJKkLlMmNnoPpQqrRSsTtUuVvWwXxYyZz")
	BitcoinAlphabet = NewAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
	IPFSAlphabet    = NewAlphabet("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
	FlickrAlphabet  = NewAlphabet("123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ")
	RippleAlphabet  = NewAlphabet("rpshnaf39wBUDNEGHJKLM4PQRST7VWXYZ2bcdeCg65jkm8oFqi1tuvAxyz")
)

// Alphabet base58字母表对象。
type Alphabet struct {
	encodeTable        [58]rune //编码后的数组
	decodeTable        [256]int //解码后的数组
	unicodeDecodeTable []rune   //
}

// NewAlphabet 会从58长度的字符串创建一个自定义字母表。 但长度必须为58为
func NewAlphabet(alphabet string) *Alphabet {
	if utf8.RuneCountInString(alphabet) != 58 {
		panic(fmt.Sprintf("Base58 Alphabet length must 58, but %d", utf8.RuneCountInString(alphabet)))
	}

	ret := new(Alphabet)
	for i := range ret.decodeTable {
		ret.decodeTable[i] = -1
	}
	ret.unicodeDecodeTable = make([]rune, 0, 58*2)
	var idx int
	var ch rune
	for _, ch = range alphabet {
		ret.encodeTable[idx] = ch
		if ch >= 0 && ch < 256 {
			ret.decodeTable[byte(ch)] = idx
		} else {
			ret.unicodeDecodeTable = append(ret.unicodeDecodeTable, ch)
			ret.unicodeDecodeTable = append(ret.unicodeDecodeTable, rune(idx))
		}
		idx++
	}
	return ret
}

// 将Alphabet字母表转换为字符串返回
func (alphabet Alphabet) String() string {
	return string(alphabet.encodeTable[:])
}

// Encode 根据传进来的参数进行加密
func Encode(input []byte, alphabet *Alphabet) string {
	// prefix 0
	inputLength := len(input)
	prefixZeroes := 0
	//剔除0开头的
	for prefixZeroes < inputLength && input[prefixZeroes] == 0 {
		prefixZeroes++
	}
	capacity := (inputLength-prefixZeroes)*138/100 + 1 // log256 / log58
	output := make([]byte, capacity)
	outputReverseEnd := capacity - 1
	var carry uint32
	var outputIdx int
	for _, inputByte := range input[prefixZeroes:] {
		carry = uint32(inputByte)

		outputIdx = capacity - 1
		for ; outputIdx > outputReverseEnd || carry != 0; outputIdx-- {
			carry += (uint32(output[outputIdx]) << 8) // XX << 8 same as: 256 * XX
			output[outputIdx] = byte(carry % 58)
			carry /= 58
		}
		outputReverseEnd = outputIdx
	}

	encodeTable := alphabet.encodeTable
	// 当不包含unicode时，使用[]byte改进性能
	if len(alphabet.unicodeDecodeTable) == 0 {
		retStrBytes := make([]byte, prefixZeroes+(capacity-1-outputReverseEnd))
		for i := 0; i < prefixZeroes; i++ {
			retStrBytes[i] = byte(encodeTable[0])
		}
		for i, n := range output[outputReverseEnd+1:] {
			retStrBytes[prefixZeroes+i] = byte(encodeTable[n])
		}
		return string(retStrBytes)
	}
	retStrRunes := make([]rune, prefixZeroes+(capacity-1-outputReverseEnd))
	for i := 0; i < prefixZeroes; i++ {
		retStrRunes[i] = encodeTable[0]
	}
	for i, n := range output[outputReverseEnd+1:] {
		retStrRunes[prefixZeroes+i] = encodeTable[n]
	}
	return string(retStrRunes)
}

// Decode 使用指定的字母表对密文进行解码
func Decode(input string, alphabet *Alphabet) ([]byte, error) {
	capacity := utf8.RuneCountInString(input)*733/1000 + 1 // log(58) / log(256)
	output := make([]byte, capacity)
	outputReverseEnd := capacity - 1
	var carry, outputIdx, i int
	var target rune

	// prefix 0
	zero58Byte := alphabet.encodeTable[0]
	prefixZeroes := 0
	skipZeros := false

	for _, target = range input {
		// collect prefix zeros
		if !skipZeros {
			if target == zero58Byte {
				prefixZeroes++
				continue
			} else {
				skipZeros = true
			}
		}

		carry = -1
		if target >= 0 && target < 256 {
			carry = alphabet.decodeTable[target]
		} else { // unicode
			for i = 0; i < len(alphabet.unicodeDecodeTable); i += 2 {
				if alphabet.unicodeDecodeTable[i] == target {
					carry = int(alphabet.unicodeDecodeTable[i+1])
					break
				}
			}
		}
		if carry == -1 {
			return nil, errors.New("invalid base58 string")
		}

		outputIdx = capacity - 1
		for ; outputIdx > outputReverseEnd || carry != 0; outputIdx-- {
			carry += 58 * int(output[outputIdx])
			output[outputIdx] = byte(uint32(carry) & 0xff) // same as: byte(uint32(carry) % 256)
			carry >>= 8                                    // same as: carry /= 256
		}
		outputReverseEnd = outputIdx
	}

	retBytes := make([]byte, prefixZeroes+(capacity-1-outputReverseEnd))
	copy(retBytes[prefixZeroes:], output[outputReverseEnd+1:])
	return retBytes, nil
}
