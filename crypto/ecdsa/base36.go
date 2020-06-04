package ecdsa

import (
	"errors"
	"math/big"
	"strconv"
	"strings"
)

var (
	Base36Chars        = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	errICAPLength      = errors.New("invalid ICAP length")
	errICAPEncoding    = errors.New("invalid ICAP encoding")
	errICAPChecksum    = errors.New("invalid ICAP checksum")
	errICAPCountryCode = errors.New("invalid ICAP country code")
	errICAPAssetIdent  = errors.New("invalid ICAP asset identifier")
	errICAPInstCode    = errors.New("invalid ICAP institution code")
	errICAPClientIdent = errors.New("invalid ICAP client identifier")
)

var (
	pn    = 3 // prefix len
	on    = 4 // orgcode len
	Big1  = big.NewInt(1)
	Big0  = big.NewInt(0)
	Big36 = big.NewInt(36)
	Big97 = big.NewInt(97)
	Big98 = big.NewInt(98)
)

func base36Encode(i *big.Int) string {
	var chars []rune
	x := new(big.Int)
	for {
		x.Mod(i, Big36)
		chars = append(chars, rune(Base36Chars[x.Uint64()]))
		i.Div(i, Big36)
		if i.Cmp(Big0) == 0 {
			break
		}
	}
	// reverse slice
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars)
}

func parseICAP(s string) (Address, error) {
	s = strings.ToUpper(s)
	if err := validCustomCheckSum(s); err != nil {
		return Address{}, err
	}
	// checksum is ISO13616, Ethereum address is base-36
	bigAddr, _ := new(big.Int).SetString(s[pn+2+on:], 36)
	return BigToAddress(bigAddr), nil
}

// validCheckSum
func validCustomCheckSum(s string) error {

	// base-36 + prefix + orgcode + checkSumNum
	s = join(s[pn+2+on:], s[:pn], s[pn+2:pn+2+on], s[pn:pn+2])
	expanded, err := iso13616Expand(s)
	if err != nil {
		return err
	}
	checkSumNum, _ := new(big.Int).SetString(expanded, 10)
	if checkSumNum.Mod(checkSumNum, Big97).Cmp(Big1) != 0 {
		return errICAPChecksum
	}
	return nil
}

func checkDigits(s, prefix, orgcode string) string {
	prefix = strings.ToUpper(prefix)
	orgcode = strings.ToUpper(orgcode)
	expanded, _ := iso13616Expand(strings.Join([]string{s, prefix, orgcode, "00"}, ""))
	num, _ := new(big.Int).SetString(expanded, 10)
	num.Sub(Big98, num.Mod(num, Big97))

	checkDigits := num.String()
	// zero padd checksum
	if len(checkDigits) == 1 {
		checkDigits = join("0", checkDigits)
	}
	return checkDigits
}

// not base-36, but expansion to decimal literal: A = 10, B = 11, ... Z = 35
func iso13616Expand(s string) (string, error) {
	var parts []string
	if !validBase36(s) {
		return "", errICAPEncoding
	}
	for _, c := range s {
		i := uint64(c)
		if i >= 65 {
			parts = append(parts, strconv.FormatUint(uint64(c)-55, 10))
		} else {
			parts = append(parts, string(c))
		}
	}
	return join(parts...), nil
}

func validBase36(s string) bool {
	for _, c := range s {
		i := uint64(c)
		// 0-9 or A-Z
		if i < 48 || (i > 57 && i < 65) || i > 90 {
			return false
		}
	}
	return true
}

func join(s ...string) string {
	return strings.Join(s, "")
}
