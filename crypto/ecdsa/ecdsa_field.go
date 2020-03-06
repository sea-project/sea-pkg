package ecdsa

// 参考文献:[HAC]:《密码学应用手册》，范奥尔斯考特，范斯通。http://cacr.uwaterloo.ca/hac/

/**
secp256k1的所有椭圆曲线运算都是在一个以256位素数为特征的有限域中完成的。
由于该精度大于可用的最大本机类型，显然需要某种形式的bignum数学。
该包实现了特定的固定精度字段算法，而不是依赖于任意精度的算术包(如math/big)来处理字段数学，因为它的大小是已知的。
因此，通过利用任意精度算法和通用模块化算法无法实现的许多优化，可以获得相当大的性能收益。
内部表示每个有限域元素的方法有很多种。例如，最明显的表示是使用一个4 uint64(64位* 4 = 256位)的数组。
然而，这种代表有几个问题。首先，当添加或相乘两个64位数字时，没有足够大的本机Go类型来处理中间结果;其次，在每个数组元素之间执行中间运算时，没有多余的空间可以溢出，这会导致昂贵的进位传播。
鉴于上述,这个实现代表字段元素10 uint32s每个单词(数组条目)视为基地2 ^ 26。这项选择的理由如下:
	1)目前大多数系统都是64位的(或者至少有64位寄存器可以用于特殊用途，比如MMX)，所以通常可以使用本机寄存器(并使用uint64s来避免额外的半字算术)来完成中间结果
	2)为了在不传播进位的情况下允许内部字的加法，每个寄存器的最大归一化值必须小于寄存器中可用位的个数
	3)由于我们处理的是32位值，所以对于#2,64位溢出是一个合理的选择
	4)需要256位的精度和属性声明在# 1,# 2,# 3,表示最好的适应这是10 uint32s基地2 ^ 26(26位* 10 = 260位,所以最后一个词只需要22位)使所需的64位(32 * 10 = 320,320 - 256 = 64)的溢出,
	因为它是如此重要的字段算法非常快高性能加密,这个包,它通常不执行任何验证。例如，一些函数只给出正确的结果，即字段是标准化的，没有检查来确保它是标准化的。
	虽然我通常更喜欢确保所有的状态和输入对于大多数包都是有效的，但是这段代码实际上只在内部使用，而且每次额外的检查都很重要。
*/

import (
	"encoding/hex"
)

// 用于使代码更具可读性的常量。
const (
	twoBitsMask   = 0x3
	fourBitsMask  = 0xf
	sixBitsMask   = 0x3f
	eightBitsMask = 0xff
)

// 与字段表示相关的常数。
const (
	//fieldWords是用于内部表示256位值的单词数。
	fieldWords = 10

	// fieldBase is the exponent used to form the numeric base of each word.
	// 2^(fieldBase*i) where i is the word position.
	fieldBase = 26

	// fieldOverflowBits is the minimum number of "overflow" bits for each
	// word in the field value.
	fieldOverflowBits = 32 - fieldBase

	// fieldBaseMask is the mask for the bits in each word needed to
	// represent the numeric base of each word (except the most significant
	// word).
	fieldBaseMask = (1 << fieldBase) - 1

	// fieldMSBBits is the number of bits in the most significant word used
	// to represent the value.
	fieldMSBBits = 256 - (fieldBase * (fieldWords - 1))

	// fieldMSBMask is the mask for the bits in the most significant word
	// needed to represent the value.
	fieldMSBMask = (1 << fieldMSBBits) - 1

	// fieldPrimeWordZero is word zero of the secp256k1 prime in the
	// internal field representation.  It is used during negation.
	fieldPrimeWordZero = 0x3fffc2f

	// fieldPrimeWordOne is word one of the secp256k1 prime in the
	// internal field representation.  It is used during negation.
	fieldPrimeWordOne = 0x3ffffbf

	// primeLowBits is the lower 2*fieldBase bits of the secp256k1 prime in
	// its standard normalized form.  It is used during modular reduction.
	primeLowBits = 0xffffefffffc2f
)

// fieldVal在secp256k1有限域上实现了优化的固定精度算法。这意味着所有的运算都是以0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc2f为模数执行的。
// 它代表每个256位值为10的32位整数在基地2 ^ 26。这为每个单词提供了6位溢出(最重要的单词中有10位)，总共提供了64位溢出(9*6 + 10 = 64)。它只实现了椭圆曲线运算所需的算法。
//下面描述了内部表示:
// 	 -----------------------------------------------------------------
// 	|        n[9]       |        n[8]       | ... |        n[0]       |
// 	| 32 bits available | 32 bits available | ... | 32 bits available |
// 	| 22 bits for value | 26 bits for value | ... | 26 bits for value |
// 	| 10 bits overflow  |  6 bits overflow  | ... |  6 bits overflow  |
// 	| Mult: 2^(26*9)    | Mult: 2^(26*8)    | ... | Mult: 2^(26*0)    |
// 	 -----------------------------------------------------------------
//
// For example, consider the number 2^49 + 1.  It would be represented as:
// 	n[0] = 1
// 	n[1] = 2^23
// 	n[2..9] = 0
//
// The full 256-bit value is then calculated by looping i from 9..0 and
// doing sum(n[i] * 2^(26i)) like so:
// 	n[9] * 2^(26*9) = 0    * 2^234 = 0
// 	n[8] * 2^(26*8) = 0    * 2^208 = 0
// 	...
// 	n[1] * 2^(26*1) = 2^23 * 2^26  = 2^49
// 	n[0] * 2^(26*0) = 1    * 2^0   = 1
// 	Sum: 0 + 0 + ... + 2^49 + 1 = 2^49 + 1
type fieldVal struct {
	n [10]uint32
}

// 将字段值作为人类可读的十六进制字符串返回。
func (f fieldVal) String() string {
	t := new(fieldVal).Set(&f).Normalize()
	return hex.EncodeToString(t.Bytes()[:])
}

// 设置字段值为0。新创建的字段值已经设置为0。此函数可用于清除现有字段值以供重用。
func (f *fieldVal) Zero() {
	f.n[0] = 0
	f.n[1] = 0
	f.n[2] = 0
	f.n[3] = 0
	f.n[4] = 0
	f.n[5] = 0
	f.n[6] = 0
	f.n[7] = 0
	f.n[8] = 0
	f.n[9] = 0
}

// 设置字段值等于传递的值。
// 返回字段值以支持链接。这支持如下语法:f:= new(fieldVal). set (f2). add(1)，以便在不修改f2的情况下，f = f2 + 1。
func (f *fieldVal) Set(val *fieldVal) *fieldVal {
	*f = *val
	return f
}

// SetInt将字段值设置为传递的整数。这是一个方便的函数，因为用小的本地整数执行一些变量是相当常见的。
// 返回字段值以支持链接。这样就可以使用f:= new(fieldVal). setint (2). mul (f2)等语法，从而使f = 2 * f2。
func (f *fieldVal) SetInt(ui uint) *fieldVal {
	f.Zero()
	f.n[0] = uint32(ui)
	return f
}

// SetBytes将传递的32字节big-endian值打包到内部字段值表示中。返回字段值以支持链接。这支持如下语法:f:= new(fieldVal). setbytes (byteArray). mul (f2)，这样f = ba * f2。
func (f *fieldVal) SetBytes(b *[32]byte) *fieldVal {
	// Pack the 256 total bits across the 10 uint32 words with a max of
	// 26-bits per word.  This could be done with a couple of for loops,
	// but this unrolled version is significantly faster.  Benchmarks show
	// this is about 34 times faster than the variant which uses loops.
	f.n[0] = uint32(b[31]) | uint32(b[30])<<8 | uint32(b[29])<<16 |
		(uint32(b[28])&twoBitsMask)<<24
	f.n[1] = uint32(b[28])>>2 | uint32(b[27])<<6 | uint32(b[26])<<14 |
		(uint32(b[25])&fourBitsMask)<<22
	f.n[2] = uint32(b[25])>>4 | uint32(b[24])<<4 | uint32(b[23])<<12 |
		(uint32(b[22])&sixBitsMask)<<20
	f.n[3] = uint32(b[22])>>6 | uint32(b[21])<<2 | uint32(b[20])<<10 |
		uint32(b[19])<<18
	f.n[4] = uint32(b[18]) | uint32(b[17])<<8 | uint32(b[16])<<16 |
		(uint32(b[15])&twoBitsMask)<<24
	f.n[5] = uint32(b[15])>>2 | uint32(b[14])<<6 | uint32(b[13])<<14 |
		(uint32(b[12])&fourBitsMask)<<22
	f.n[6] = uint32(b[12])>>4 | uint32(b[11])<<4 | uint32(b[10])<<12 |
		(uint32(b[9])&sixBitsMask)<<20
	f.n[7] = uint32(b[9])>>6 | uint32(b[8])<<2 | uint32(b[7])<<10 |
		uint32(b[6])<<18
	f.n[8] = uint32(b[5]) | uint32(b[4])<<8 | uint32(b[3])<<16 |
		(uint32(b[2])&twoBitsMask)<<24
	f.n[9] = uint32(b[2])>>2 | uint32(b[1])<<6 | uint32(b[0])<<14
	return f
}

// SetByteSlice将传递的big-endian值打包到内部字段值表示中。只使用前32字节。
// 因此，由调用方决定是否使用适当大小的号码，否则将截断该值。返回字段值以支持链接。这支持以下语法:f:= new(fieldVal).SetByteSlice(byteSlice)
func (f *fieldVal) SetByteSlice(b []byte) *fieldVal {
	var b32 [32]byte
	for i := 0; i < len(b); i++ {
		if i < 32 {
			b32[i+(32-len(b))] = b[i]
		}
	}
	return f.SetBytes(&b32)
}

// SetHex将传递的big-endian十六进制字符串解码到内部字段值表示中。只使用前32字节。
// 返回字段值以支持链接。这支持如下语法:f:= new(fieldVal).SetHex("0abc").Add(1)以便f = 0x0abc + 1
func (f *fieldVal) SetHex(hexString string) *fieldVal {
	if len(hexString)%2 != 0 {
		hexString = "0" + hexString
	}
	bytes, _ := hex.DecodeString(hexString)
	return f.SetByteSlice(bytes)
}

// 将内部字段词归一化，归一化到期望范围内，利用素数的特殊形式对secp256k1素数进行快速模约。
func (f *fieldVal) Normalize() *fieldVal {
	// 字段表示在每个单词中留下6位溢出，因此可以执行中间计算，而不需要在计算期间将进位传播到每个更高的单词。为了标准化，首先我们需要向右“压缩”完整的256位值，并将其余的64位作为大小。
	m := f.n[0]
	t0 := m & fieldBaseMask
	m = (m >> fieldBase) + f.n[1]
	t1 := m & fieldBaseMask
	m = (m >> fieldBase) + f.n[2]
	t2 := m & fieldBaseMask
	m = (m >> fieldBase) + f.n[3]
	t3 := m & fieldBaseMask
	m = (m >> fieldBase) + f.n[4]
	t4 := m & fieldBaseMask
	m = (m >> fieldBase) + f.n[5]
	t5 := m & fieldBaseMask
	m = (m >> fieldBase) + f.n[6]
	t6 := m & fieldBaseMask
	m = (m >> fieldBase) + f.n[7]
	t7 := m & fieldBaseMask
	m = (m >> fieldBase) + f.n[8]
	t8 := m & fieldBaseMask
	m = (m >> fieldBase) + f.n[9]
	t9 := m & fieldMSBMask
	m = m >> fieldMSBBits

	// 此时，如果大小大于0，则整体值大于可能的最大256位值。特别是，它比最大值“大多少倍”。由于这个字段是对secp256k1 '做算术模，我们需要对'执行模约。
	//
	// 每(HAC)部分14.3.4:还原法特殊形式的模,模量时的特殊形式m = b ^ t - c,可以实现高效的减少。
	//
	// The secp256k1 prime is equivalent to 2^256 - 4294968273, so it fits
	// this criteria.
	//
	// 4294968273 in field representation (base 2^26) is:
	// n[0] = 977
	// n[1] = 64
	// That is to say (2^26 * 64) + 977 = 4294968273
	//
	// 在参考部分中给出的算法通常会重复，直到商为零。然而，由于我们的字段表示，我们已经知道至少需要重复多少次，因为它是当前m中的值。
	// 因此，我们可以简单地将大小乘以素数的字段表示，然后进行一次迭代。
	// 注意，当大小为0时，不会有任何变化，所以在这种情况下，我们可以跳过这个，但是无论如何总是运行，都允许它在恒定的时间内运行。
	r := t0 + m*977
	t0 = r & fieldBaseMask
	r = (r >> fieldBase) + t1 + m*64
	t1 = r & fieldBaseMask
	r = (r >> fieldBase) + t2
	t2 = r & fieldBaseMask
	r = (r >> fieldBase) + t3
	t3 = r & fieldBaseMask
	r = (r >> fieldBase) + t4
	t4 = r & fieldBaseMask
	r = (r >> fieldBase) + t5
	t5 = r & fieldBaseMask
	r = (r >> fieldBase) + t6
	t6 = r & fieldBaseMask
	r = (r >> fieldBase) + t7
	t7 = r & fieldBaseMask
	r = (r >> fieldBase) + t8
	t8 = r & fieldBaseMask
	r = (r >> fieldBase) + t9
	t9 = r & fieldMSBMask

	// 在这一点上,结果将是在0 < =结果< = ' + (2 ^ 64 - c)。因此,可能需要一个减法的'如果当前结果大于或等于'。下面对常数时间做最后的约简。
	// 注意，这里的if/else有意按位执行操作，或者使用0执行操作，即使它不会更改值以确保分支之间的时间是恒定的。
	var mask int32
	lowBits := uint64(t1)<<fieldBase | uint64(t0)
	if lowBits < primeLowBits {
		mask |= -1
	} else {
		mask |= 0
	}
	if t2 < fieldBaseMask {
		mask |= -1
	} else {
		mask |= 0
	}
	if t3 < fieldBaseMask {
		mask |= -1
	} else {
		mask |= 0
	}
	if t4 < fieldBaseMask {
		mask |= -1
	} else {
		mask |= 0
	}
	if t5 < fieldBaseMask {
		mask |= -1
	} else {
		mask |= 0
	}
	if t6 < fieldBaseMask {
		mask |= -1
	} else {
		mask |= 0
	}
	if t7 < fieldBaseMask {
		mask |= -1
	} else {
		mask |= 0
	}
	if t8 < fieldBaseMask {
		mask |= -1
	} else {
		mask |= 0
	}
	if t9 < fieldMSBMask {
		mask |= -1
	} else {
		mask |= 0
	}
	lowBits -= ^uint64(mask) & primeLowBits
	t0 = uint32(lowBits & fieldBaseMask)
	t1 = uint32((lowBits >> fieldBase) & fieldBaseMask)
	t2 = t2 & uint32(mask)
	t3 = t3 & uint32(mask)
	t4 = t4 & uint32(mask)
	t5 = t5 & uint32(mask)
	t6 = t6 & uint32(mask)
	t7 = t7 & uint32(mask)
	t8 = t8 & uint32(mask)
	t9 = t9 & uint32(mask)

	// 最后，设置归一化和约简后的单词。
	f.n[0] = t0
	f.n[1] = t1
	f.n[2] = t2
	f.n[3] = t3
	f.n[4] = t4
	f.n[5] = t5
	f.n[6] = t6
	f.n[7] = t7
	f.n[8] = t8
	f.n[9] = t9
	return f
}

// PutBytes使用传递的字节数组将字段值解压缩为一个32字节的大端值。还有一个类似的函数Bytes，它将字段值解压缩到一个新的数组中并返回该数组。这个版本是提供的，因为允许调用者重用缓冲区可以减少分配的数量。
//
// 这个函数必须对字段值进行规范化才能返回正确的结果。
func (f *fieldVal) PutBytes(b *[32]byte) {
	// 从10个uint32单词中解压缩256位，每个单词最多26位。这可以通过几个for循环来完成，但是这个展开的版本更快一些。基准测试显示这比使用循环的版本快10倍。
	b[31] = byte(f.n[0] & eightBitsMask)
	b[30] = byte((f.n[0] >> 8) & eightBitsMask)
	b[29] = byte((f.n[0] >> 16) & eightBitsMask)
	b[28] = byte((f.n[0]>>24)&twoBitsMask | (f.n[1]&sixBitsMask)<<2)
	b[27] = byte((f.n[1] >> 6) & eightBitsMask)
	b[26] = byte((f.n[1] >> 14) & eightBitsMask)
	b[25] = byte((f.n[1]>>22)&fourBitsMask | (f.n[2]&fourBitsMask)<<4)
	b[24] = byte((f.n[2] >> 4) & eightBitsMask)
	b[23] = byte((f.n[2] >> 12) & eightBitsMask)
	b[22] = byte((f.n[2]>>20)&sixBitsMask | (f.n[3]&twoBitsMask)<<6)
	b[21] = byte((f.n[3] >> 2) & eightBitsMask)
	b[20] = byte((f.n[3] >> 10) & eightBitsMask)
	b[19] = byte((f.n[3] >> 18) & eightBitsMask)
	b[18] = byte(f.n[4] & eightBitsMask)
	b[17] = byte((f.n[4] >> 8) & eightBitsMask)
	b[16] = byte((f.n[4] >> 16) & eightBitsMask)
	b[15] = byte((f.n[4]>>24)&twoBitsMask | (f.n[5]&sixBitsMask)<<2)
	b[14] = byte((f.n[5] >> 6) & eightBitsMask)
	b[13] = byte((f.n[5] >> 14) & eightBitsMask)
	b[12] = byte((f.n[5]>>22)&fourBitsMask | (f.n[6]&fourBitsMask)<<4)
	b[11] = byte((f.n[6] >> 4) & eightBitsMask)
	b[10] = byte((f.n[6] >> 12) & eightBitsMask)
	b[9] = byte((f.n[6]>>20)&sixBitsMask | (f.n[7]&twoBitsMask)<<6)
	b[8] = byte((f.n[7] >> 2) & eightBitsMask)
	b[7] = byte((f.n[7] >> 10) & eightBitsMask)
	b[6] = byte((f.n[7] >> 18) & eightBitsMask)
	b[5] = byte(f.n[8] & eightBitsMask)
	b[4] = byte((f.n[8] >> 8) & eightBitsMask)
	b[3] = byte((f.n[8] >> 16) & eightBitsMask)
	b[2] = byte((f.n[8]>>24)&twoBitsMask | (f.n[9]&sixBitsMask)<<2)
	b[1] = byte((f.n[9] >> 6) & eightBitsMask)
	b[0] = byte((f.n[9] >> 14) & eightBitsMask)
}

//Bytes将字段值解压缩为32字节的大端值。有关允许传递缓冲区的变体，请参阅PutBytes，这有助于通过允许调用者重用缓冲区来减少分配的数量。
//这个函数必须对字段值进行规范化才能返回正确的结果。
func (f *fieldVal) Bytes() *[32]byte {
	b := new([32]byte)
	f.PutBytes(b)
	return b
}

// is0返回字段值是否等于0。
func (f *fieldVal) IsZero() bool {
	// 只有在没有设置任何位的情况下，该值才能为零。这是一个固定时间的实现。
	bits := f.n[0] | f.n[1] | f.n[2] | f.n[3] | f.n[4] |
		f.n[5] | f.n[6] | f.n[7] | f.n[8] | f.n[9]

	return bits == 0
}

// 判断IsOdd返回字段值是否为奇数。
//
// 这个函数必须对字段值进行规范化才能返回正确的结果。
func (f *fieldVal) IsOdd() bool {
	// 只有奇数具有底部位集。
	return f.n[0]&1 == 1
}

// =返回两个字段值是否相同。要使这个函数返回正确的结果，必须对正在比较的两个字段值进行规范化。
func (f *fieldVal) Equals(val *fieldVal) bool {
	// Xor只在不同的位置设置位，所以只有在对每个单词进行xoring后没有设置位时，两个字段值才能相同。这是一个固定时间的实现。
	bits := (f.n[0] ^ val.n[0]) | (f.n[1] ^ val.n[1]) | (f.n[2] ^ val.n[2]) |
		(f.n[3] ^ val.n[3]) | (f.n[4] ^ val.n[4]) | (f.n[5] ^ val.n[5]) |
		(f.n[6] ^ val.n[6]) | (f.n[7] ^ val.n[7]) | (f.n[8] ^ val.n[8]) |
		(f.n[9] ^ val.n[9])

	return bits == 0
}

// NegateVal否定传递的值并将结果存储在f中。调用者必须为正确的结果提供传递值的大小。
//返回字段值以支持链接。这支持如下语法:f. negateval (f2). addint(1)使f = -f2 + 1。
func (f *fieldVal) NegateVal(val *fieldVal, magnitude uint32) *fieldVal {
	// 否定的是' -值.  但是，为了允许对字段值进行否定而不需要首先对其进行规范化/减少，可以将其乘以要调整的大小(即它与规范化值之间的“距离”)。
	// 此外，由于对一个值求负值会使它与归一化范围相差一个数量级，因此需要添加1来进行补偿。

	// 为了直观理解，假设您正在执行对12取余的运算(画一个时钟)，并且您正在对数字7求反。所以你从12开始(对12取余当然是0)然后倒数7次到5。
	// 注意这里是12-7 = 5。现在，假设你从19开始，这个数已经大于模量，等于7(模12)当一个值已经在期望范围内时，它的大小是1。
	// 因为19是一个额外的“步骤”，它的大小(mod 12)是2。因为模的任何倍数都是零(mod m)，答案可以通过简单地用模的模和减去模的模来进行变换。根据这个例子，这是(2*12)-19 = 5。
	f.n[0] = (magnitude+1)*fieldPrimeWordZero - val.n[0]
	f.n[1] = (magnitude+1)*fieldPrimeWordOne - val.n[1]
	f.n[2] = (magnitude+1)*fieldBaseMask - val.n[2]
	f.n[3] = (magnitude+1)*fieldBaseMask - val.n[3]
	f.n[4] = (magnitude+1)*fieldBaseMask - val.n[4]
	f.n[5] = (magnitude+1)*fieldBaseMask - val.n[5]
	f.n[6] = (magnitude+1)*fieldBaseMask - val.n[6]
	f.n[7] = (magnitude+1)*fieldBaseMask - val.n[7]
	f.n[8] = (magnitude+1)*fieldBaseMask - val.n[8]
	f.n[9] = (magnitude+1)*fieldMSBMask - val.n[9]

	return f
}

// 否定字段值。修改现有字段值。调用方必须为正确的结果提供字段值的大小。
// 返回字段值以支持链接。这样就可以使用以下语法:f. negate (). addint(1)，使f = -f + 1。
func (f *fieldVal) Negate(magnitude uint32) *fieldVal {
	return f.NegateVal(f, magnitude)
}

// AddInt将传递的整数添加到现有的字段值，并将结果存储在f中。这是一个方便的函数，因为使用小的本地整数执行一些变量是相当常见的。
//返回字段值以支持链接。这支持以下语法:f. addint (1). add (f2)使f = f + 1 + f2。
func (f *fieldVal) AddInt(ui uint) *fieldVal {
	// 由于字段表示有意提供溢出位，因此可以使用无carryless加法，因为进位是单词的安全部分，将被规范化。
	f.n[0] += uint32(ui)

	return f
}

// Add将传递的值添加到现有字段值，并将结果存储在f中。
//
// 返回字段值以支持链接。这支持如下语法:f. add (f2). addint(1)，使f = f + f2 + 1。
func (f *fieldVal) Add(val *fieldVal) *fieldVal {
	// 由于字段表示有意提供溢出位，因此可以使用无carryless加法，因为进位是每个单词的安全部分，将被规范化。这显然可以在循环中完成，但是展开的版本更快。
	f.n[0] += val.n[0]
	f.n[1] += val.n[1]
	f.n[2] += val.n[2]
	f.n[3] += val.n[3]
	f.n[4] += val.n[4]
	f.n[5] += val.n[5]
	f.n[6] += val.n[6]
	f.n[7] += val.n[7]
	f.n[8] += val.n[8]
	f.n[9] += val.n[9]

	return f
}

// Add2将传递的两个字段值相加，并将结果存储在f中。
//
// 返回字段值以支持链接。这就启用了像f3这样的语法。ad2 (f, f2)。addint(1)使得f3 = f + f2 + 1。
func (f *fieldVal) Add2(val *fieldVal, val2 *fieldVal) *fieldVal {
	// 由于字段表示有意提供溢出位，因此可以使用无carryless加法，因为进位是每个单词的安全部分，将被规范化。这显然可以在循环中完成，但是展开的版本更快。
	f.n[0] = val.n[0] + val2.n[0]
	f.n[1] = val.n[1] + val2.n[1]
	f.n[2] = val.n[2] + val2.n[2]
	f.n[3] = val.n[3] + val2.n[3]
	f.n[4] = val.n[4] + val2.n[4]
	f.n[5] = val.n[5] + val2.n[5]
	f.n[6] = val.n[6] + val2.n[6]
	f.n[7] = val.n[7] + val2.n[7]
	f.n[8] = val.n[8] + val2.n[8]
	f.n[9] = val.n[9] + val2.n[9]

	return f
}

// MulInt将字段值乘以传递的int，并将结果存储在f中。注意，如果将该值乘以任何单个单词超过了最大uint32，则该函数可能溢出。因此，调用方必须确保在使用此函数之前不会发生溢出。
// 返回字段值以支持链接。这支持如下语法: f.MulInt(2).Add(f2) so that f = 2 * f + f2.
func (f *fieldVal) MulInt(val uint) *fieldVal {
	// 由于字段表示的每个单词都可以保留到fieldOverflowBits的额外位，这些位将被规范化，因此可以安全地将每个单词相乘，而不使用更大的类型或进行传播，只要值不会溢出uint32。这显然可以在循环中完成，但是展开的版本更快。
	ui := uint32(val)
	f.n[0] *= ui
	f.n[1] *= ui
	f.n[2] *= ui
	f.n[3] *= ui
	f.n[4] *= ui
	f.n[5] *= ui
	f.n[6] *= ui
	f.n[7] *= ui
	f.n[8] *= ui
	f.n[9] *= ui

	return f
}

// Mul将传递的值乘以现有字段值，并将结果存储在f中。请注意，如果任何单个单词的乘法值超过最大uint32，则该函数可能溢出。在实践中，这意味着乘法中两个值的大小都必须是8的最大值。
//
// 返回字段值以支持链接。这支持以下语法: f.Mul(f2).AddInt(1) so that f = (f * f2) + 1.
func (f *fieldVal) Mul(val *fieldVal) *fieldVal {
	return f.Mul2(f, val)
}

// Mul2将传递的两个字段值相乘，并将结果存储在f中。请注意，如果单个单词的乘积超过uint32，这个函数就会溢出。在实践中，这意味着乘法中两个值的大小都必须是8的最大值。
//
// 返回字段值以支持链接。这支持以下语法: f3.Mul2(f, f2).AddInt(1) so that f3 = (f * f2) + 1.
func (f *fieldVal) Mul2(val *fieldVal, val2 *fieldVal) *fieldVal {
	// 这可以通过几个for循环和一个存储中间项的数组来完成，但是这个展开的版本要快得多。

	// Terms for 2^(fieldBase*0).
	m := uint64(val.n[0]) * uint64(val2.n[0])
	t0 := m & fieldBaseMask

	// Terms for 2^(fieldBase*1).
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[1]) +
		uint64(val.n[1])*uint64(val2.n[0])
	t1 := m & fieldBaseMask

	// Terms for 2^(fieldBase*2).
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[2]) +
		uint64(val.n[1])*uint64(val2.n[1]) +
		uint64(val.n[2])*uint64(val2.n[0])
	t2 := m & fieldBaseMask

	// Terms for 2^(fieldBase*3).
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[3]) +
		uint64(val.n[1])*uint64(val2.n[2]) +
		uint64(val.n[2])*uint64(val2.n[1]) +
		uint64(val.n[3])*uint64(val2.n[0])
	t3 := m & fieldBaseMask

	// Terms for 2^(fieldBase*4).
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[4]) +
		uint64(val.n[1])*uint64(val2.n[3]) +
		uint64(val.n[2])*uint64(val2.n[2]) +
		uint64(val.n[3])*uint64(val2.n[1]) +
		uint64(val.n[4])*uint64(val2.n[0])
	t4 := m & fieldBaseMask

	// Terms for 2^(fieldBase*5).
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[5]) +
		uint64(val.n[1])*uint64(val2.n[4]) +
		uint64(val.n[2])*uint64(val2.n[3]) +
		uint64(val.n[3])*uint64(val2.n[2]) +
		uint64(val.n[4])*uint64(val2.n[1]) +
		uint64(val.n[5])*uint64(val2.n[0])
	t5 := m & fieldBaseMask

	// Terms for 2^(fieldBase*6).
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[6]) +
		uint64(val.n[1])*uint64(val2.n[5]) +
		uint64(val.n[2])*uint64(val2.n[4]) +
		uint64(val.n[3])*uint64(val2.n[3]) +
		uint64(val.n[4])*uint64(val2.n[2]) +
		uint64(val.n[5])*uint64(val2.n[1]) +
		uint64(val.n[6])*uint64(val2.n[0])
	t6 := m & fieldBaseMask

	// Terms for 2^(fieldBase*7).
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[7]) +
		uint64(val.n[1])*uint64(val2.n[6]) +
		uint64(val.n[2])*uint64(val2.n[5]) +
		uint64(val.n[3])*uint64(val2.n[4]) +
		uint64(val.n[4])*uint64(val2.n[3]) +
		uint64(val.n[5])*uint64(val2.n[2]) +
		uint64(val.n[6])*uint64(val2.n[1]) +
		uint64(val.n[7])*uint64(val2.n[0])
	t7 := m & fieldBaseMask

	// Terms for 2^(fieldBase*8).
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[8]) +
		uint64(val.n[1])*uint64(val2.n[7]) +
		uint64(val.n[2])*uint64(val2.n[6]) +
		uint64(val.n[3])*uint64(val2.n[5]) +
		uint64(val.n[4])*uint64(val2.n[4]) +
		uint64(val.n[5])*uint64(val2.n[3]) +
		uint64(val.n[6])*uint64(val2.n[2]) +
		uint64(val.n[7])*uint64(val2.n[1]) +
		uint64(val.n[8])*uint64(val2.n[0])
	t8 := m & fieldBaseMask

	// Terms for 2^(fieldBase*9).
	m = (m >> fieldBase) +
		uint64(val.n[0])*uint64(val2.n[9]) +
		uint64(val.n[1])*uint64(val2.n[8]) +
		uint64(val.n[2])*uint64(val2.n[7]) +
		uint64(val.n[3])*uint64(val2.n[6]) +
		uint64(val.n[4])*uint64(val2.n[5]) +
		uint64(val.n[5])*uint64(val2.n[4]) +
		uint64(val.n[6])*uint64(val2.n[3]) +
		uint64(val.n[7])*uint64(val2.n[2]) +
		uint64(val.n[8])*uint64(val2.n[1]) +
		uint64(val.n[9])*uint64(val2.n[0])
	t9 := m & fieldBaseMask

	// Terms for 2^(fieldBase*10).
	m = (m >> fieldBase) +
		uint64(val.n[1])*uint64(val2.n[9]) +
		uint64(val.n[2])*uint64(val2.n[8]) +
		uint64(val.n[3])*uint64(val2.n[7]) +
		uint64(val.n[4])*uint64(val2.n[6]) +
		uint64(val.n[5])*uint64(val2.n[5]) +
		uint64(val.n[6])*uint64(val2.n[4]) +
		uint64(val.n[7])*uint64(val2.n[3]) +
		uint64(val.n[8])*uint64(val2.n[2]) +
		uint64(val.n[9])*uint64(val2.n[1])
	t10 := m & fieldBaseMask

	// Terms for 2^(fieldBase*11).
	m = (m >> fieldBase) +
		uint64(val.n[2])*uint64(val2.n[9]) +
		uint64(val.n[3])*uint64(val2.n[8]) +
		uint64(val.n[4])*uint64(val2.n[7]) +
		uint64(val.n[5])*uint64(val2.n[6]) +
		uint64(val.n[6])*uint64(val2.n[5]) +
		uint64(val.n[7])*uint64(val2.n[4]) +
		uint64(val.n[8])*uint64(val2.n[3]) +
		uint64(val.n[9])*uint64(val2.n[2])
	t11 := m & fieldBaseMask

	// Terms for 2^(fieldBase*12).
	m = (m >> fieldBase) +
		uint64(val.n[3])*uint64(val2.n[9]) +
		uint64(val.n[4])*uint64(val2.n[8]) +
		uint64(val.n[5])*uint64(val2.n[7]) +
		uint64(val.n[6])*uint64(val2.n[6]) +
		uint64(val.n[7])*uint64(val2.n[5]) +
		uint64(val.n[8])*uint64(val2.n[4]) +
		uint64(val.n[9])*uint64(val2.n[3])
	t12 := m & fieldBaseMask

	// Terms for 2^(fieldBase*13).
	m = (m >> fieldBase) +
		uint64(val.n[4])*uint64(val2.n[9]) +
		uint64(val.n[5])*uint64(val2.n[8]) +
		uint64(val.n[6])*uint64(val2.n[7]) +
		uint64(val.n[7])*uint64(val2.n[6]) +
		uint64(val.n[8])*uint64(val2.n[5]) +
		uint64(val.n[9])*uint64(val2.n[4])
	t13 := m & fieldBaseMask

	// Terms for 2^(fieldBase*14).
	m = (m >> fieldBase) +
		uint64(val.n[5])*uint64(val2.n[9]) +
		uint64(val.n[6])*uint64(val2.n[8]) +
		uint64(val.n[7])*uint64(val2.n[7]) +
		uint64(val.n[8])*uint64(val2.n[6]) +
		uint64(val.n[9])*uint64(val2.n[5])
	t14 := m & fieldBaseMask

	// Terms for 2^(fieldBase*15).
	m = (m >> fieldBase) +
		uint64(val.n[6])*uint64(val2.n[9]) +
		uint64(val.n[7])*uint64(val2.n[8]) +
		uint64(val.n[8])*uint64(val2.n[7]) +
		uint64(val.n[9])*uint64(val2.n[6])
	t15 := m & fieldBaseMask

	// Terms for 2^(fieldBase*16).
	m = (m >> fieldBase) +
		uint64(val.n[7])*uint64(val2.n[9]) +
		uint64(val.n[8])*uint64(val2.n[8]) +
		uint64(val.n[9])*uint64(val2.n[7])
	t16 := m & fieldBaseMask

	// Terms for 2^(fieldBase*17).
	m = (m >> fieldBase) +
		uint64(val.n[8])*uint64(val2.n[9]) +
		uint64(val.n[9])*uint64(val2.n[8])
	t17 := m & fieldBaseMask

	// Terms for 2^(fieldBase*18).
	m = (m >> fieldBase) + uint64(val.n[9])*uint64(val2.n[9])
	t18 := m & fieldBaseMask

	// What's left is for 2^(fieldBase*19).
	t19 := m >> fieldBase

	// 此时，所有术语都被分组到各自的基中.
	//
	// 每(HAC)部分14.3.4:还原法特殊形式的模,模量时的特殊形式m = b ^ t - c,可以实现高效的减少所提供的算法。
	//
	// secp256k1 '相当于2 ^ 256 - 4294968273,这符合标准。
	//
	// 4294968273 in field representation (base 2^26) is:
	// n[0] = 977
	// n[1] = 64
	// That is to say (2^26 * 64) + 977 = 4294968273
	//
	// 因为每个单词是在基地26,上面的条款(t10和)从260位(与256位的最终期望的范围),所以从上面“c”的字段表示需要调整为额外的4位乘以2 ^ 4 = 16。4294968273 * 16 = 68719492368。因此，“c”的调整字段表示为:
	// n[0] = 977 * 16 = 15632
	// n[1] = 64 * 16 = 1024
	// That is to say (2^26 * 1024) + 15632 = 68719492368
	//
	// 为了减少最后一项t19，需要整个“c”值，而不仅仅是n[0]，因为没有更多的项可以处理n[1]。这意味着上面的位可能还剩下一些大小在下面处理。
	m = t0 + t10*15632
	t0 = m & fieldBaseMask
	m = (m >> fieldBase) + t1 + t10*1024 + t11*15632
	t1 = m & fieldBaseMask
	m = (m >> fieldBase) + t2 + t11*1024 + t12*15632
	t2 = m & fieldBaseMask
	m = (m >> fieldBase) + t3 + t12*1024 + t13*15632
	t3 = m & fieldBaseMask
	m = (m >> fieldBase) + t4 + t13*1024 + t14*15632
	t4 = m & fieldBaseMask
	m = (m >> fieldBase) + t5 + t14*1024 + t15*15632
	t5 = m & fieldBaseMask
	m = (m >> fieldBase) + t6 + t15*1024 + t16*15632
	t6 = m & fieldBaseMask
	m = (m >> fieldBase) + t7 + t16*1024 + t17*15632
	t7 = m & fieldBaseMask
	m = (m >> fieldBase) + t8 + t17*1024 + t18*15632
	t8 = m & fieldBaseMask
	m = (m >> fieldBase) + t9 + t18*1024 + t19*68719492368
	t9 = m & fieldMSBMask
	m = m >> fieldMSBBits

	// 此时，如果大小大于0，则整体值大于可能的最大256位值。特别是，它比最大值“大多少倍”。
	//
	// [HAC] 14.3.4节中提出的算法重复执行，直到商为零。然而，由于上述原因，我们已经知道至少需要重复多少次，因为它是当前m中的值。
	// 因此，我们可以简单地将大小乘以素数的字段表示，然后进行一次迭代。注意，当大小为0时，不会有任何变化，所以在这种情况下，我们可以跳过这个，但是无论如何总是运行，
	// 都允许它在恒定的时间内运行。最终结果将在范围0 < =结果< = ' +(2 ^ 64 - c),所以它是保证有一个1级,但它是不正常。
	d := t0 + m*977
	f.n[0] = uint32(d & fieldBaseMask)
	d = (d >> fieldBase) + t1 + m*64
	f.n[1] = uint32(d & fieldBaseMask)
	f.n[2] = uint32((d >> fieldBase) + t2)
	f.n[3] = uint32(t3)
	f.n[4] = uint32(t4)
	f.n[5] = uint32(t5)
	f.n[6] = uint32(t6)
	f.n[7] = uint32(t7)
	f.n[8] = uint32(t8)
	f.n[9] = uint32(t9)

	return f
}

// 平方等于字段值的平方。修改现有字段值。注意，如果任何单个单词的乘法运算超过了最大uint32，那么该函数就会溢出。在实践中，这意味着字段的大小必须是8，以防止溢出。
// 返回字段值以支持链接。这支持以下语法: f.Square().Mul(f2) so that f = f^2 * f2.
func (f *fieldVal) Square() *fieldVal {
	return f.SquareVal(f)
}

// 对传递的值进行平方，并将结果存储在f中。注意，如果任何单个单词的乘法超过了最大uint32，则该函数可能溢出。在实践中，这意味着被squred字段的大小必须为8，以防止溢出。
//
// 返回字段值以支持链接。这支持以下语法: f3.SquareVal(f).Mul(f) so that f3 = f^2 * f = f^3.
func (f *fieldVal) SquareVal(val *fieldVal) *fieldVal {
	// 这可以通过几个for循环和一个存储中间项的数组来完成，但是这个展开的版本要快得多。

	// Terms for 2^(fieldBase*0).
	m := uint64(val.n[0]) * uint64(val.n[0])
	t0 := m & fieldBaseMask

	// Terms for 2^(fieldBase*1).
	m = (m >> fieldBase) + 2*uint64(val.n[0])*uint64(val.n[1])
	t1 := m & fieldBaseMask

	// Terms for 2^(fieldBase*2).
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[2]) +
		uint64(val.n[1])*uint64(val.n[1])
	t2 := m & fieldBaseMask

	// Terms for 2^(fieldBase*3).
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[3]) +
		2*uint64(val.n[1])*uint64(val.n[2])
	t3 := m & fieldBaseMask

	// Terms for 2^(fieldBase*4).
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[4]) +
		2*uint64(val.n[1])*uint64(val.n[3]) +
		uint64(val.n[2])*uint64(val.n[2])
	t4 := m & fieldBaseMask

	// Terms for 2^(fieldBase*5).
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[5]) +
		2*uint64(val.n[1])*uint64(val.n[4]) +
		2*uint64(val.n[2])*uint64(val.n[3])
	t5 := m & fieldBaseMask

	// Terms for 2^(fieldBase*6).
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[6]) +
		2*uint64(val.n[1])*uint64(val.n[5]) +
		2*uint64(val.n[2])*uint64(val.n[4]) +
		uint64(val.n[3])*uint64(val.n[3])
	t6 := m & fieldBaseMask

	// Terms for 2^(fieldBase*7).
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[7]) +
		2*uint64(val.n[1])*uint64(val.n[6]) +
		2*uint64(val.n[2])*uint64(val.n[5]) +
		2*uint64(val.n[3])*uint64(val.n[4])
	t7 := m & fieldBaseMask

	// Terms for 2^(fieldBase*8).
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[8]) +
		2*uint64(val.n[1])*uint64(val.n[7]) +
		2*uint64(val.n[2])*uint64(val.n[6]) +
		2*uint64(val.n[3])*uint64(val.n[5]) +
		uint64(val.n[4])*uint64(val.n[4])
	t8 := m & fieldBaseMask

	// Terms for 2^(fieldBase*9).
	m = (m >> fieldBase) +
		2*uint64(val.n[0])*uint64(val.n[9]) +
		2*uint64(val.n[1])*uint64(val.n[8]) +
		2*uint64(val.n[2])*uint64(val.n[7]) +
		2*uint64(val.n[3])*uint64(val.n[6]) +
		2*uint64(val.n[4])*uint64(val.n[5])
	t9 := m & fieldBaseMask

	// Terms for 2^(fieldBase*10).
	m = (m >> fieldBase) +
		2*uint64(val.n[1])*uint64(val.n[9]) +
		2*uint64(val.n[2])*uint64(val.n[8]) +
		2*uint64(val.n[3])*uint64(val.n[7]) +
		2*uint64(val.n[4])*uint64(val.n[6]) +
		uint64(val.n[5])*uint64(val.n[5])
	t10 := m & fieldBaseMask

	// Terms for 2^(fieldBase*11).
	m = (m >> fieldBase) +
		2*uint64(val.n[2])*uint64(val.n[9]) +
		2*uint64(val.n[3])*uint64(val.n[8]) +
		2*uint64(val.n[4])*uint64(val.n[7]) +
		2*uint64(val.n[5])*uint64(val.n[6])
	t11 := m & fieldBaseMask

	// Terms for 2^(fieldBase*12).
	m = (m >> fieldBase) +
		2*uint64(val.n[3])*uint64(val.n[9]) +
		2*uint64(val.n[4])*uint64(val.n[8]) +
		2*uint64(val.n[5])*uint64(val.n[7]) +
		uint64(val.n[6])*uint64(val.n[6])
	t12 := m & fieldBaseMask

	// Terms for 2^(fieldBase*13).
	m = (m >> fieldBase) +
		2*uint64(val.n[4])*uint64(val.n[9]) +
		2*uint64(val.n[5])*uint64(val.n[8]) +
		2*uint64(val.n[6])*uint64(val.n[7])
	t13 := m & fieldBaseMask

	// Terms for 2^(fieldBase*14).
	m = (m >> fieldBase) +
		2*uint64(val.n[5])*uint64(val.n[9]) +
		2*uint64(val.n[6])*uint64(val.n[8]) +
		uint64(val.n[7])*uint64(val.n[7])
	t14 := m & fieldBaseMask

	// Terms for 2^(fieldBase*15).
	m = (m >> fieldBase) +
		2*uint64(val.n[6])*uint64(val.n[9]) +
		2*uint64(val.n[7])*uint64(val.n[8])
	t15 := m & fieldBaseMask

	// Terms for 2^(fieldBase*16).
	m = (m >> fieldBase) +
		2*uint64(val.n[7])*uint64(val.n[9]) +
		uint64(val.n[8])*uint64(val.n[8])
	t16 := m & fieldBaseMask

	// Terms for 2^(fieldBase*17).
	m = (m >> fieldBase) + 2*uint64(val.n[8])*uint64(val.n[9])
	t17 := m & fieldBaseMask

	// Terms for 2^(fieldBase*18).
	m = (m >> fieldBase) + uint64(val.n[9])*uint64(val.n[9])
	t18 := m & fieldBaseMask

	// What's left is for 2^(fieldBase*19).
	t19 := m >> fieldBase

	m = t0 + t10*15632
	t0 = m & fieldBaseMask
	m = (m >> fieldBase) + t1 + t10*1024 + t11*15632
	t1 = m & fieldBaseMask
	m = (m >> fieldBase) + t2 + t11*1024 + t12*15632
	t2 = m & fieldBaseMask
	m = (m >> fieldBase) + t3 + t12*1024 + t13*15632
	t3 = m & fieldBaseMask
	m = (m >> fieldBase) + t4 + t13*1024 + t14*15632
	t4 = m & fieldBaseMask
	m = (m >> fieldBase) + t5 + t14*1024 + t15*15632
	t5 = m & fieldBaseMask
	m = (m >> fieldBase) + t6 + t15*1024 + t16*15632
	t6 = m & fieldBaseMask
	m = (m >> fieldBase) + t7 + t16*1024 + t17*15632
	t7 = m & fieldBaseMask
	m = (m >> fieldBase) + t8 + t17*1024 + t18*15632
	t8 = m & fieldBaseMask
	m = (m >> fieldBase) + t9 + t18*1024 + t19*68719492368
	t9 = m & fieldMSBMask
	m = m >> fieldMSBBits

	// 此时，如果大小大于0，则整体值大于可能的最大256位值。特别是，它比最大值“大多少倍”。
	//
	// [HAC] 14.3.4节中提出的算法重复执行，直到商为零。然而，由于上述原因，我们已经知道至少需要重复多少次，因为它是当前m中的值。
	// 因此，我们可以简单地将大小乘以素数的字段表示，然后进行一次迭代。注意，当大小为0时，不会有任何变化，所以在这种情况下，我们可以跳过这个
	// ，但是无论如何总是运行，都允许它在恒定的时间内运行。最终结果将在范围0 < =结果< = ' +(2 ^ 64 - c),所以它是保证有一个1级,但它是不正常。
	n := t0 + m*977
	f.n[0] = uint32(n & fieldBaseMask)
	n = (n >> fieldBase) + t1 + m*64
	f.n[1] = uint32(n & fieldBaseMask)
	f.n[2] = uint32((n >> fieldBase) + t2)
	f.n[3] = uint32(t3)
	f.n[4] = uint32(t4)
	f.n[5] = uint32(t5)
	f.n[6] = uint32(t6)
	f.n[7] = uint32(t7)
	f.n[8] = uint32(t8)
	f.n[9] = uint32(t9)

	return f
}

// 求该域值的模乘逆。修改现有字段值。
// 返回字段值以支持链接。这支持以下语法: f.Inverse().Mul(f2) so that f = f^-1 * f2.
func (f *fieldVal) Inverse() *fieldVal {
	// 费马小定理指出,对于一个非零的数,' ' p,一个^(p - 1)= 1 p(mod)。由于multipliciative逆a * b = 1(mod p),此前,b = * ^(p 2)= ^(p - 1)= 1 p(mod)。因此,一个^(p 2)乘法逆元。
	//
	// 为了有效地计算一个^ (p 2), p 2需要分成一系列的广场和multipications乘法的数量降至最低,需要比平方(因为它们更贵)。中间结果也被保存和重用。
	//
	// The secp256k1 prime - 2 is 2^256 - 4294968275.
	//
	// 这需要258个场的平方和33个场的乘法。
	var a2, a3, a4, a10, a11, a21, a42, a45, a63, a1019, a1023 fieldVal
	a2.SquareVal(f)
	a3.Mul2(&a2, f)
	a4.SquareVal(&a2)
	a10.SquareVal(&a4).Mul(&a2)
	a11.Mul2(&a10, f)
	a21.Mul2(&a10, &a11)
	a42.SquareVal(&a21)
	a45.Mul2(&a42, &a3)
	a63.Mul2(&a42, &a21)
	a1019.SquareVal(&a63).Square().Square().Square().Mul(&a11)
	a1023.Mul2(&a1019, &a4)
	f.Set(&a63)                                    // f = a^(2^6 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^11 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^16 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^16 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^21 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^26 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^26 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^31 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^36 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^36 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^41 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^46 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^46 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^51 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^56 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^56 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^61 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^66 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^66 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^71 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^76 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^76 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^81 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^86 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^86 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^91 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^96 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^96 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^101 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^106 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^106 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^111 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^116 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^116 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^121 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^126 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^126 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^131 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^136 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^136 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^141 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^146 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^146 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^151 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^156 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^156 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^161 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^166 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^166 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^171 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^176 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^176 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^181 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^186 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^186 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^191 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^196 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^196 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^201 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^206 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^206 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^211 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^216 - 1024)
	f.Mul(&a1023)                                  // f = a^(2^216 - 1)
	f.Square().Square().Square().Square().Square() // f = a^(2^221 - 32)
	f.Square().Square().Square().Square().Square() // f = a^(2^226 - 1024)
	f.Mul(&a1019)                                  // f = a^(2^226 - 5)
	f.Square().Square().Square().Square().Square() // f = a^(2^231 - 160)
	f.Square().Square().Square().Square().Square() // f = a^(2^236 - 5120)
	f.Mul(&a1023)                                  // f = a^(2^236 - 4097)
	f.Square().Square().Square().Square().Square() // f = a^(2^241 - 131104)
	f.Square().Square().Square().Square().Square() // f = a^(2^246 - 4195328)
	f.Mul(&a1023)                                  // f = a^(2^246 - 4194305)
	f.Square().Square().Square().Square().Square() // f = a^(2^251 - 134217760)
	f.Square().Square().Square().Square().Square() // f = a^(2^256 - 4294968320)
	return f.Mul(&a45)                             // f = a^(2^256 - 4294968275) = a^(p-2)
}
