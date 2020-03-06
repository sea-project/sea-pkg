package sha3

import (
	"crypto"
	"hash"
	"io"
)

// 海绵方向表示字节流过海绵的方向。
type spongeDirection int

func init() {
	crypto.RegisterHash(crypto.SHA3_224, New224)
	crypto.RegisterHash(crypto.SHA3_256, New256)
	crypto.RegisterHash(crypto.SHA3_384, New384)
	crypto.RegisterHash(crypto.SHA3_512, New512)
}

const (
	// 海绵吸收表明海绵吸收输入。
	spongeAbsorbing spongeDirection = iota

	// 海绵挤压表明,海绵被挤压。
	spongeSqueezing

	// maxRate是内部缓冲区的最大大小。SHAKE-256目前需要最大的缓冲区。
	maxRate = 168
)

type state struct {
	// 普通海绵组件。
	a    [25]uint64 // 哈希的主要状态
	buf  []byte     // 点到存储
	rate int        // 要使用的状态字节数

	dsbyte  byte
	storage [maxRate]byte

	// 针对SHA-3和SHAKE。
	outputLen int             // 默认输出大小(以字节为单位)
	state     spongeDirection // 海绵是吸收还是挤压
}

// 返回该哈希函数下的海绵速率。
func (d *state) BlockSize() int { return d.rate }

// 返回哈希函数的输出大小(以字节为单位)
func (d *state) Size() int { return d.outputLen }

// 通过对海绵状态和字节缓冲区进行归零，并设置海绵来清除内部状态。吸收状态
func (d *state) Reset() {
	// Zero the permutation's state.
	for i := range d.a {
		d.a[i] = 0
	}
	d.state = spongeAbsorbing
	d.buf = d.storage[:0]
}

func (d *state) clone() *state {
	ret := *d
	if ret.state == spongeAbsorbing {
		ret.buf = ret.storage[:len(ret.buf)]
	} else {
		ret.buf = ret.storage[d.rate-cap(d.buf) : d.rate]
	}

	return &ret
}

// 应用KeccakF-1600置换。它处理任何输入输出缓冲
func (d *state) permute() {
	switch d.state {
	case spongeAbsorbing:
		// 如果我们吸收,我们需要xor应用置换之前的输入状态。.
		xorIn(d, d.buf)
		d.buf = d.storage[:0]
		keccakF1600(&d.a)
	case spongeSqueezing:
		// 如果我们挤压,我们需要应用permutatin之前复制更多的产出。
		keccakF1600(&d.a)
		d.buf = d.storage[:d.rate]
		copyOut(d, d.buf)
	}
}

// pad在dsbyte中追加域分隔位，应用多比特率10..填充规则，并打乱状态
func (d *state) padAndPermute(dsbyte byte) {
	if d.buf == nil {
		d.buf = d.storage[:0]
	}

	d.buf = append(d.buf, dsbyte)
	zerosStart := len(d.buf)
	d.buf = d.storage[:d.rate]
	for i := zerosStart; i < d.rate; i++ {
		d.buf[i] = 0
	}

	d.buf[d.rate-1] ^= 0x80
	// Apply the permutation
	d.permute()
	d.state = spongeSqueezing
	d.buf = d.storage[:d.rate]
	copyOut(d, d.buf)
}

// 将更多的数据写到哈希的状态中。如果在写入之后向ShakeHash写入更多数据，则会产生错误
func (d *state) Write(p []byte) (written int, err error) {
	if d.state != spongeAbsorbing {
		panic("sha3: write to sponge after read")
	}
	if d.buf == nil {
		d.buf = d.storage[:0]
	}
	written = len(p)

	for len(p) > 0 {
		if len(d.buf) == 0 && len(p) >= d.rate {
			// 快速路径;吸收输入的全部“速率”字节并应用排列。
			xorIn(d, p[:d.rate])
			p = p[d.rate:]
			keccakF1600(&d.a)
		} else {
			// 缓慢的路径;缓冲输入，直到我们可以填充海绵，然后xor它。
			todo := d.rate - len(d.buf)
			if todo > len(p) {
				todo = len(p)
			}
			d.buf = append(d.buf, p[:todo]...)
			p = p[todo:]

			// 如果海绵是满的，应用排列。
			if len(d.buf) == d.rate {
				d.permute()
			}
		}
	}

	return
}

// Read从海绵中挤压任意数量的字节
func (d *state) Read(out []byte) (n int, err error) {

	if d.state == spongeAbsorbing {
		d.padAndPermute(d.dsbyte)
	}

	n = len(out)

	for len(out) > 0 {
		n := copy(out, d.buf)
		d.buf = d.buf[n:]
		out = out[n:]

		if len(d.buf) == 0 {
			d.permute()
		}
	}

	return
}

// Sum对哈希状态应用填充，然后挤压出所需的输出字节数
func (d *state) Sum(in []byte) []byte {
	dup := d.clone()
	hash := make([]byte, dup.outputLen)
	dup.Read(hash)
	return append(in, hash...)
}

// ShakeHash 定义了支持任意长度输出的哈希函数的接口。
// --------------------------------------下面这些接口定义了ShakeHash接口，并提供了创建SHAKE实例的函数，以及将字节散列到任意长度输出的实用函数。--------------------------------------//
type ShakeHash interface {

	// 写将更多的数据吸收到哈希的状态中。如果在读取输出之后将输入写到它，它会报错。
	io.Writer

	// Read从散列读取更多输出;读取会影响哈希的状态。(ShakeHash。因此，Read与Hash.Sum非常不同)它从不返回错误。
	io.Reader

	// 克隆返回当前状态下的ShakeHash副本。
	Clone() ShakeHash

	// Reset将ShakeHash重置为初始状态.
	Reset()
}

func (d *state) Clone() ShakeHash {
	return d.clone()
}

// NewShake128 创建一个新的SHAKE128可变输出长度ShakeHash。如果使用至少32字节的输出，它的一般安全强度是128位，可以抵御所有攻击。
func NewShake128() ShakeHash { return &state{rate: 168, dsbyte: 0x1f} }

// NewShake256 创建一个新的SHAKE128可变输出长度ShakeHash。如果使用至少64字节的输出，它的一般安全强度是256位，可以抵御所有攻击。
func NewShake256() ShakeHash { return &state{rate: 136, dsbyte: 0x1f} }

// ShakeSum128 将任意长度的数据摘要写入哈希。
func ShakeSum128(hash, data []byte) {
	h := NewShake128()
	h.Write(data)
	h.Read(hash)
}

// ShakeSum256 将任意长度的数据摘要写入哈希。
func ShakeSum256(hash, data []byte) {
	h := NewShake256()
	h.Write(data)
	h.Read(hash)
}

// NewKeccak256 创建一个新的Keccak-256散列。参与了大部分工作，详情求见具体篇
//--------------------------------------下面这些接口提供了用于创建SHA-3和SHAKE散列函数实例的函数，以及用于散列字节的实用函数。--------------------------------------//
func NewKeccak256() hash.Hash { return &state{rate: 136, outputLen: 32, dsbyte: 0x01} }

// NewKeccak512 创建一个新的Keccak-512散列。参与了创建验证缓存和检查一个块是否满足PoW难度要求，要么使用通常的ethash缓存，要么使用完整的DAG使远程挖掘更快
func NewKeccak512() hash.Hash { return &state{rate: 72, outputLen: 64, dsbyte: 0x01} }

// New224 创建一个新的SHA3-224散列。它的一般安全强度是224位对前图像攻击，112位对碰撞攻击。
func New224() hash.Hash { return &state{rate: 144, outputLen: 28, dsbyte: 0x06} }

// New256 创建一个新的SHA3-256散列。它的一般安全强度是256位的图像攻击，和128位的碰撞攻击。
func New256() hash.Hash { return &state{rate: 136, outputLen: 32, dsbyte: 0x06} }

// New384 创建一个新的SHA3-384散列。它的一般安全强度是384位对前映像攻击，192位对碰撞攻击。
func New384() hash.Hash { return &state{rate: 104, outputLen: 48, dsbyte: 0x06} }

// New512 创建一个新的SHA3-512散列。它的一般安全强度是512位，以防止预映像攻击，和256位，以防止碰撞攻击
func New512() hash.Hash { return &state{rate: 72, outputLen: 64, dsbyte: 0x06} }

// Sum224 返回数据的SHA3-224摘要。
func Sum224(data []byte) (digest [28]byte) {
	h := New224()
	h.Write(data)
	h.Sum(digest[:0])
	return
}

// Sum256 返回数据的SHA3-256摘要。
func Sum256(data []byte) (digest [32]byte) {
	h := New256()
	h.Write(data)
	h.Sum(digest[:0])
	return
}

// Sum384 返回数据的SHA3-384摘要。
func Sum384(data []byte) (digest [48]byte) {
	h := New384()
	h.Write(data)
	h.Sum(digest[:0])
	return
}

// Sum512 返回数据的SHA3-512摘要。
func Sum512(data []byte) (digest [64]byte) {
	h := New512()
	h.Write(data)
	h.Sum(digest[:0])
	return
}

// Keccak256 使用sha3 256加密内容
func Keccak256(data ...[]byte) []byte {
	d := NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}
