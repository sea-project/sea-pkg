package ecdsa

import (
	"bytes"
	e "crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"math/big"
)

var (

	// 用于调整ECDSA延展性的曲线顺序和halforder
	order     = new(big.Int).Set(S256().N)
	halforder = new(big.Int).Rsh(order, 1)

	// 用于RFC6979实现时测试nonce的正确性
	one = big.NewInt(1)

	// 用于用字节0x01填充字节片。这里提供它是为了避免多次创建它。
	oneInitializer = []byte{0x01}

	// 返回的错误规范填充。
	errNegativeValue          = errors.New("value may be interpreted as negative")
	errExcessivelyPaddedValue = errors.New("value is excessively padded")
)

// Signature
type Signature struct {
	R *big.Int
	S *big.Int
}

// Verify 通过调用ecdsa的公钥来验证哈希的签名是否正确
func (sig *Signature) Verify(hash []byte, pubKey *PublicKey) bool {
	return e.Verify(pubKey.ToECDSA(), hash, sig.R, sig.S)
}

// IsEqual 比较两个签名是否相等
func (sig *Signature) IsEqual(otherSig *Signature) bool {
	return sig.R.Cmp(otherSig.R) == 0 && sig.S.Cmp(otherSig.S) == 0
}

// SignCompact 使用指定的koblitz曲线上的指定私钥从而在散列中生成数据的压缩签名
func SignCompact(curve *KoblitzCurve, key *PrivateKey, hash []byte, isCompressedKey bool) ([]byte, error) {
	sig, err := key.Sign(hash)
	if err != nil {
		return nil, err
	}

	// bitcoind检查R和S的位长。ecdsa签名算法返回R和S mod N，因此它们是曲线的位大小，因此大小正确。
	for i := 0; i < (curve.H+1)*2; i++ {
		pk, err := recoverKeyFromSignature(curve, sig, hash, i, true)
		if err == nil && pk.X.Cmp(key.X) == 0 && pk.Y.Cmp(key.Y) == 0 {
			result := make([]byte, 1, 2*curve.byteSize+1)
			result[0] = 27 + byte(i)
			if isCompressedKey {
				result[0] += 4
			}

			curvelen := (curve.BitSize + 7) / 8

			bytelen := (sig.R.BitLen() + 7) / 8
			if bytelen < curvelen {
				result = append(result,
					make([]byte, curvelen-bytelen)...)
			}
			result = append(result, sig.R.Bytes()...)

			bytelen = (sig.S.BitLen() + 7) / 8
			if bytelen < curvelen {
				result = append(result,
					make([]byte, curvelen-bytelen)...)
			}
			result = append(result, sig.S.Bytes()...)

			return result, nil
		}
	}
	return nil, errors.New("no valid solution for pubkey found")
}

// RecoverCompact 验证“曲线”中Koblitz曲线的“哈希”的“压缩签名”并推出公钥
func RecoverCompact(curve *KoblitzCurve, signature, hash []byte) (*PublicKey, bool, error) {
	bitlen := (curve.BitSize + 7) / 8
	if len(signature) != 1+bitlen*2 {
		return nil, false, errors.New("invalid compact signature size")
	}

	iteration := int((signature[0] - 27) & ^byte(4))

	sig := &Signature{
		R: new(big.Int).SetBytes(signature[1 : bitlen+1]),
		S: new(big.Int).SetBytes(signature[bitlen+1:]),
	}
	key, err := recoverKeyFromSignature(curve, sig, hash, iteration, false)
	if err != nil {
		return nil, false, err
	}

	return key, ((signature[0] - 27) & 4) == 4, nil
}

// 从给定消息散列“msg”上的签名“sig”中提取公钥
func recoverKeyFromSignature(curve *KoblitzCurve, sig *Signature, msg []byte,
	iter int, doChecks bool) (*PublicKey, error) {

	Rx := new(big.Int).Mul(curve.Params().N,
		new(big.Int).SetInt64(int64(iter/2)))
	Rx.Add(Rx, sig.R)
	if Rx.Cmp(curve.Params().P) != -1 {
		return nil, errors.New("calculated Rx is larger than curve P")
	}

	Ry, err := decompressPoint(curve, Rx, iter%2 == 1)
	if err != nil {
		return nil, err
	}

	if doChecks {
		nRx, nRy := curve.ScalarMult(Rx, Ry, curve.Params().N.Bytes())
		if nRx.Sign() != 0 || nRy.Sign() != 0 {
			return nil, errors.New("n*R does not equal the point at infinity")
		}
	}

	e := hashToInt(msg, curve)

	invr := new(big.Int).ModInverse(sig.R, curve.Params().N)

	invrS := new(big.Int).Mul(invr, sig.S)
	invrS.Mod(invrS, curve.Params().N)
	sRx, sRy := curve.ScalarMult(Rx, Ry, invrS.Bytes())

	e.Neg(e)
	e.Mod(e, curve.Params().N)
	e.Mul(e, invr)
	e.Mod(e, curve.Params().N)
	minuseGx, minuseGy := curve.ScalarBaseMult(e.Bytes())

	Qx, Qy := curve.Add(sRx, sRy, minuseGx, minuseGy)

	return &PublicKey{
		Curve: curve,
		X:     Qx,
		Y:     Qy,
	}, nil
}

// decompressPoint 在给定的X点和要使用的解的情况下，解压给定曲线上的一个点。
func decompressPoint(curve *KoblitzCurve, x *big.Int, ybit bool) (*big.Int, error) {
	// TODO: This will probably only work for secp256k1 due to
	// optimizations.

	// Y = +-sqrt(x^3 + B)
	x3 := new(big.Int).Mul(x, x)
	x3.Mul(x3, x)
	x3.Add(x3, curve.Params().B)

	// //现在计算sqrt mod p (x2 + B)这个代码用于基于tonelli/shanks做一个完整的sqrt，但是它被https://bitcointalk.org/index.php?topic=162805.msg1712294#msg1712294中引用的算法所替代
	y := new(big.Int).Exp(x3, curve.q, curve.Params().P)

	if ybit != isOdd(y) {
		y.Sub(curve.Params().P, y)
	}
	if ybit != isOdd(y) {
		return nil, fmt.Errorf("ybit doesn't match oddness")
	}
	return y, nil
}

func isOdd(a *big.Int) bool {
	return a.Bit(0) == 1
}

// signRFC6979根据RFC6979和BIP 62生成一个确定的ECDSA签名
func signRFC6979(privateKey *PrivateKey, hash []byte) (*Signature, error) {

	privkey := privateKey
	N := order
	k := nonceRFC6979(privkey.D, hash)
	inv := new(big.Int).ModInverse(k, N)
	r, _ := privkey.Curve.ScalarBaseMult(k.Bytes())
	if r.Cmp(N) == 1 {
		r.Sub(r, N)
	}

	if r.Sign() == 0 {
		return nil, errors.New("calculated R is zero")
	}

	e := hashToInt(hash, privkey.Curve)
	s := new(big.Int).Mul(privkey.D, r)
	s.Add(s, e)
	s.Mul(s, inv)
	s.Mod(s, N)

	if s.Cmp(halforder) == 1 {
		s.Sub(N, s)
	}
	if s.Sign() == 0 {
		return nil, errors.New("calculated S is zero")
	}
	return &Signature{R: r, S: s}, nil
}

// 生成一个ECDSA nonce (' k ')。它以一个32字节的散列作为输入，并立即返回32字节，以便在ECDSA算法中使用。
func nonceRFC6979(privkey *big.Int, hash []byte) *big.Int {

	curve := S256()
	q := curve.Params().N
	x := privkey
	alg := sha256.New

	qlen := q.BitLen()
	holen := alg().Size()
	rolen := (qlen + 7) >> 3
	bx := append(int2octets(x, rolen), bits2octets(hash, curve, rolen)...)

	v := bytes.Repeat(oneInitializer, holen)

	// Go将所有分配的内存归零
	k := make([]byte, holen)

	k = mac(alg, k, append(append(v, 0x00), bx...))

	v = mac(alg, k, v)

	k = mac(alg, k, append(append(v, 0x01), bx...))

	v = mac(alg, k, v)

	for {
		var t []byte

		for len(t)*8 < qlen {
			v = mac(alg, k, v)
			t = append(t, v...)
		}

		secret := hashToInt(t, curve)
		if secret.Cmp(one) >= 0 && secret.Cmp(q) < 0 {
			return secret
		}
		k = mac(alg, k, append(v, 0x00))
		v = mac(alg, k, v)
	}
}

// hashToInt将哈希值转换为整数
func hashToInt(hash []byte, c elliptic.Curve) *big.Int {
	orderBits := c.Params().N.BitLen()
	orderBytes := (orderBits + 7) / 8
	if len(hash) > orderBytes {
		hash = hash[:orderBytes]
	}

	ret := new(big.Int).SetBytes(hash)
	excess := len(hash)*8 - orderBits
	if excess > 0 {
		ret.Rsh(ret, uint(excess))
	}
	return ret
}

func int2octets(v *big.Int, rolen int) []byte {
	out := v.Bytes()

	// left pad with zeros if it's too short
	if len(out) < rolen {
		out2 := make([]byte, rolen)
		copy(out2[rolen-len(out):], out)
		return out2
	}

	// drop most significant bytes if it's too long
	if len(out) > rolen {
		out2 := make([]byte, rolen)
		copy(out2, out[len(out)-rolen:])
		return out2
	}

	return out
}

func bits2octets(in []byte, curve elliptic.Curve, rolen int) []byte {
	z1 := hashToInt(in, curve)
	z2 := new(big.Int).Sub(z1, curve.Params().N)
	if z2.Sign() < 0 {
		return int2octets(z1, rolen)
	}
	return int2octets(z2, rolen)
}

// mac返回给定键和消息的HMAC
func mac(alg func() hash.Hash, k, m []byte) []byte {
	h := hmac.New(alg, k)
	h.Write(m)
	return h.Sum(nil)
}
