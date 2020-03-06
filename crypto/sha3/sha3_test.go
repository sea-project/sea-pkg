package sha3

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"hash"
	"os"
	"strings"
	"testing"
)

const (
	testString  = "brekeccakkeccak koax koax"
	katFilename = "testdata/keccakKats.json.deflate"
)

// 通过不同的hash散列函数创建不同位数的示例,它们都有相同的规律
func Test_Example_Flow2(t *testing.T) {
	buf := []byte("演示NewKeccak256()散列函数生成Hash，它的outputLen位数为32位,但它生成的Hash为64位")
	keccak256 := NewKeccak256()
	keccak256.Write(buf)
	md1 := keccak256.Sum(nil)
	s := hex.EncodeToString(md1)
	t.Log(s)

	buf2 := []byte(s)
	keccak512 := NewKeccak512()
	keccak512.Write(buf2)
	md2 := keccak512.Sum(nil)
	s2 := hex.EncodeToString(md2)
	t.Log(s2)

	buf3 := []byte(s2)
	new224 := New224()
	new224.Write(buf3)
	md3 := new224.Sum(nil)
	t.Log(hex.EncodeToString(md3))
}

//200000000	         9.64 ns/op
func BenchmarkNewKeccak256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewKeccak256()
	}
}

//200000000	         9.85 ns/op
func BenchmarkNewKeccak512(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewKeccak512()
	}
}

// 通过不同的hash散列函数创建不同位数的示例,它们都有相同的规律
func Test_Example_Flow(t *testing.T) {
	buf := []byte("演示NewKeccak256()散列函数生成Hash，它的outputLen位数为32位,但它生成的Hash为64位")
	keccak256 := NewKeccak256()
	keccak256.Write(buf)
	md1 := keccak256.Sum(nil)
	t.Log(hex.EncodeToString(md1))

	buf2 := []byte("演示NewKeccak512()散列函数生成Hash，它的outputLen位数为64位,但它生成的Hash为128位")
	keccak512 := NewKeccak512()
	keccak512.Write(buf2)
	md2 := keccak512.Sum(nil)
	t.Log(hex.EncodeToString(md2))

	buf3 := []byte("演示New224()散列函数生成Hash，它的outputLen位数为28位,但它生成的Hash为56位")
	new224 := New224()
	new224.Write(buf3)
	md3 := new224.Sum(nil)
	t.Log(hex.EncodeToString(md3))
}

//500000	      2580 ns/op
func Benchmark_Sum(b *testing.B) {
	buf := []byte("演示NewKeccak256()散列函数生成Hash，它的outputLen位数为32位,但它生成的Hash为64位")
	for i := 0; i < b.N; i++ {
		keccak256 := NewKeccak256()
		keccak256.Write(buf)
		md1 := keccak256.Sum(nil)
		b.Log(hex.EncodeToString(md1))
	}
}

// SHAKE的内部使用实例，用于对KATs进行测试。
func newHashShake128() hash.Hash {
	return &state{rate: 168, dsbyte: 0x1f, outputLen: 512}
}
func newHashShake256() hash.Hash {
	return &state{rate: 136, dsbyte: 0x1f, outputLen: 512}
}

//testdigest包含返回散列的函数。输出长度等于SHA-3和SHAKE实例的KAT长度的散列实例。
var testDigests = map[string]func() hash.Hash{
	"SHA3-224": New224,
	"SHA3-256": New256,
	"SHA3-384": New384,
	"SHA3-512": New512,
	"SHAKE128": newHashShake128,
	"SHAKE256": newHashShake256,
}

// testShakes包含返回ShakeHash实例的函数，用于测试特定于ShakeHash的接口。
var testShakes = map[string]func() ShakeHash{
	"SHAKE128": NewShake128,
	"SHAKE256": NewShake256,
}

// 用于编组JSON测试用例的结构。
type KeccakKats struct {
	Kats map[string][]struct {
		Digest  string `json:"digest"`
		Length  int64  `json:"length"`
		Message string `json:"message"`
	}
}

func testUnalignedAndGeneric(t *testing.T, testf func(impl string)) {
	xorInOrig, copyOutOrig := xorIn, copyOut
	xorIn, copyOut = testXorInGeneric, testCopyOutGeneric
	testf("generic")
	if xorImplementationUnaligned != "generic" {
		xorIn, copyOut = testXorInGeneric, testCopyOutGeneric
		testf("unaligned")
	}
	xorIn, copyOut = xorInOrig, copyOutOrig
}

// copyOutGeneric将ulint64s复制到字节缓冲区。
func testCopyOutGeneric(d *state, b []byte) {
	for i := 0; len(b) >= 8; i++ {
		binary.LittleEndian.PutUint64(b, d.a[i])
		b = b[8:]
	}
}

// xorInGeneric将buf中的字节转换为状态;它没有对内存布局或对齐做出不可移植的假设。
func testXorInGeneric(d *state, buf []byte) {
	n := len(buf) / 8

	for i := 0; i < n; i++ {
		a := binary.LittleEndian.Uint64(buf)
		d.a[i] ^= a
		buf = buf[8:]
	}
}

// 测试SHA-3和Shake实现(测试向量存储在keccakKats.json.deflate中，因为它们的长度不同)。
func TestKeccakKats(t *testing.T) {
	testUnalignedAndGeneric(t, func(impl string) {
		// Read the KATs.
		deflated, err := os.Open(katFilename)
		if err != nil {
			t.Errorf("error opening %s: %s", katFilename, err)
		}
		file := flate.NewReader(deflated)
		dec := json.NewDecoder(file)
		var katSet KeccakKats
		err = dec.Decode(&katSet)
		if err != nil {
			t.Errorf("error decoding KATs: %s", err)
		}

		// Do the KATs.
		for functionName, kats := range katSet.Kats {
			d := testDigests[functionName]()
			for _, kat := range kats {
				d.Reset()
				in, err := hex.DecodeString(kat.Message)
				if err != nil {
					t.Errorf("error decoding KAT: %s", err)
				}
				d.Write(in[:kat.Length/8])
				got := strings.ToUpper(hex.EncodeToString(d.Sum(nil)))
				if got != kat.Digest {
					t.Errorf("function=%s, implementation=%s, length=%d\nmessage:\n  %s\ngot:\n  %s\nwanted:\n %s",
						functionName, impl, kat.Length, kat.Message, got, kat.Digest)
					t.Logf("wanted %+v", kat)
					t.FailNow()
				}
				continue
			}
		}
	})
}

// testunnedwrite测试将数据写入具有较小输入缓冲区的任意模式。
func TestUnalignedWrite(t *testing.T) {
	testUnalignedAndGeneric(t, func(impl string) {
		buf := sequentialBytes(0x10000)
		for alg, df := range testDigests {
			d := df()
			d.Reset()
			d.Write(buf)
			want := d.Sum(nil)
			d.Reset()
			for i := 0; i < len(buf); {
				//循环偏移量，偏移量形成137字节的序列。
				//因为137是质数，所以这个序列应该适用于所有的角情况。
				offsets := [17]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 1}
				for _, j := range offsets {
					if v := len(buf) - i; v < j {
						j = v
					}
					d.Write(buf[i : i+j])
					i += j
				}
			}
			got := d.Sum(nil)
			if !bytes.Equal(got, want) {
				t.Errorf("Unaligned writes, implementation=%s, alg=%s\ngot %q, want %q", impl, alg, got, want)
			}
		}
	})
}

// TestAppend测试在需要重新分配时追加是否有效。
func TestAppend(t *testing.T) {
	testUnalignedAndGeneric(t, func(impl string) {
		d := New224()

		for capacity := 2; capacity <= 66; capacity += 64 {
			// 第一次循环时，Sum必须重新分配。
			// 第二次，不会了。
			buf := make([]byte, 2, capacity)
			d.Reset()
			d.Write([]byte{0xcc})
			buf = d.Sum(buf)
			expected := "0000DF70ADC49B2E76EEE3A6931B93FA41841C3AF2CDF5B32A18B5478C39"
			if got := strings.ToUpper(hex.EncodeToString(buf)); got != expected {
				t.Errorf("got %s, want %s", got, expected)
			}
		}
	})
}
func TestNew256(t *testing.T) {

	buf := []byte("set(uint256)")
	keccak256 := NewKeccak256()
	keccak256.Write(buf)
	md1 := keccak256.Sum(nil)
	t.Log(hex.EncodeToString(md1))

}

// TestAppendNoRealloc测试在不需要重新分配的情况下追加是否有效.
func TestAppendNoRealloc(t *testing.T) {
	testUnalignedAndGeneric(t, func(impl string) {
		buf := make([]byte, 1, 200)
		d := New224()
		d.Write([]byte{0xcc})
		buf = d.Sum(buf)
		expected := "00DF70ADC49B2E76EEE3A6931B93FA41841C3AF2CDF5B32A18B5478C39"
		if got := strings.ToUpper(hex.EncodeToString(buf)); got != expected {
			t.Errorf("%s: got %s, want %s", impl, got, expected)
		}
	})
}

// 检查一次压缩完整输出是否与重复压缩实例产生相同的输出。
func TestSqueezing(t *testing.T) {
	testUnalignedAndGeneric(t, func(impl string) {
		for functionName, newShakeHash := range testShakes {
			d0 := newShakeHash()
			d0.Write([]byte(testString))
			ref := make([]byte, 32)
			d0.Read(ref)

			d1 := newShakeHash()
			d1.Write([]byte(testString))
			var multiple []byte
			for range ref {
				one := make([]byte, 1)
				d1.Read(one)
				multiple = append(multiple, one...)
			}
			if !bytes.Equal(ref, multiple) {
				t.Errorf("%s (%s): squeezing %d bytes one at a time failed", functionName, impl, len(ref))
			}
			t.Log()
		}
	})
}

// sequentialBytes产生一个大小为连续字节0x00、0x01、…，用于测试。
func sequentialBytes(size int) []byte {
	result := make([]byte, size)
	for i := range result {
		result[i] = byte(i)
	}
	return result
}

// 压测在没有输入数据的情况下测量置换函数的速度。 5000000	       457 ns/op	 437.09 MB/s
func BenchmarkPermutationFunction(b *testing.B) {
	b.SetBytes(int64(200))
	var lanes [25]uint64
	for i := 0; i < b.N; i++ {
		keccakF1600(&lanes)
	}
}

//200000000	         9.76 ns/op
func BenchmarkNew224(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New224()
	}
}

//200000000	         9.63 ns/op
func BenchmarkNew256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New256()
	}
}

//100000000	        10.6 ns/op
func BenchmarkNew384(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New384()
	}
}

//100000000	        10.5 ns/op
func BenchmarkNew512(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New512()
	}
}

// 测试对每个buflen的num缓冲区进行哈希的速度。
func benchmarkHash(b *testing.B, h hash.Hash, size, num int) {
	b.StopTimer()
	h.Reset()
	data := sequentialBytes(size)
	b.SetBytes(int64(size * num))
	b.StartTimer()

	var state []byte
	for i := 0; i < b.N; i++ {
		for j := 0; j < num; j++ {
			h.Write(data)
		}
		state = h.Sum(state[:0])
	}
	b.StopTimer()
	h.Reset()
}

// 专门用于Shake实例，在读取输出时不需要副本。
func benchmarkShake(b *testing.B, h ShakeHash, size, num int) {
	b.StopTimer()
	h.Reset()
	data := sequentialBytes(size)
	d := make([]byte, 32)

	b.SetBytes(int64(size * num))
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		h.Reset()
		for j := 0; j < num; j++ {
			h.Write(data)
		}
		h.Read(d)
	}
}

func BenchmarkSha3_512_MTU(b *testing.B)  { benchmarkHash(b, New512(), 1350, 1) }          // 200000	      8796 ns/op	 153.47 MB/s
func BenchmarkSha3_384_MTU(b *testing.B)  { benchmarkHash(b, New384(), 1350, 1) }          // 200000	      6776 ns/op	 199.21 MB/s
func BenchmarkSha3_256_MTU(b *testing.B)  { benchmarkHash(b, New256(), 1350, 1) }          // 300000	      5116 ns/op	 263.86 MB/s
func BenchmarkSha3_224_MTU(b *testing.B)  { benchmarkHash(b, New224(), 1350, 1) }          // 300000	      4448 ns/op	 303.50 MB/s
func BenchmarkShake128_MTU(b *testing.B)  { benchmarkShake(b, NewShake128(), 1350, 1) }    // 500000	      3967 ns/op	 340.27 MB/s
func BenchmarkShake256_MTU(b *testing.B)  { benchmarkShake(b, NewShake256(), 1350, 1) }    // 300000	      4158 ns/op	 324.61 MB/s
func BenchmarkShake256_16x(b *testing.B)  { benchmarkShake(b, NewShake256(), 16, 1024) }   // 20000	     68317 ns/op	 239.82 MB/s
func BenchmarkShake256_1MiB(b *testing.B) { benchmarkShake(b, NewShake256(), 1024, 1024) } // 500	   3229306 ns/op	 324.71 MB/s
func BenchmarkSha3_512_1MiB(b *testing.B) { benchmarkHash(b, New512(), 1024, 1024) }       //200	   7589689 ns/op	 138.16 MB/s

func Test_Example_sum(t *testing.T) {
	buf := []byte("some data to hash sdasdasdasdasdasdas")
	// 哈希需要64字节长才能具有256位的抗碰撞能力。
	h := make([]byte, 64)
	t.Log("写入前:", h)
	// 计算buf的64字节哈希并将其放入h。
	Sum384(buf)
	got := strings.ToUpper(hex.EncodeToString(buf))
	t.Log("写入后:", got)
}
