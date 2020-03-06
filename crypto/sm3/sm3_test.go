package sm3

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func byteToString(b []byte) string {
	ret := ""
	for i := 0; i < len(b); i++ {
		ret += fmt.Sprintf("%02x", b[i])
	}
	fmt.Println("ret = ", ret)
	return ret
}
func TestSm3(t *testing.T) {
	msg := []byte("test")
	err := ioutil.WriteFile("ifile", msg, os.FileMode(0644)) // 生成测试文件
	if err != nil {
		log.Fatal(err)
	}
	msg, err = ioutil.ReadFile("ifile")
	if err != nil {
		log.Fatal(err)
	}
	hw := New()
	hw.Write(msg)
	hash := hw.Sum(nil)
	fmt.Println(hash)
	fmt.Printf("hash = %d\n", len(hash))
	fmt.Printf("%s\n", byteToString(hash))
	hash1 := Sm3Sum(msg)
	fmt.Println(hash1)
	fmt.Printf("%s\n", byteToString(hash1))

}

//1000000	      1814 ns/op	     304 B/op	      10 allocs/op
func BenchmarkSm3(t *testing.B) {
	t.ReportAllocs()
	msg := []byte("test")
	hw := New()
	for i := 0; i < t.N; i++ {

		hw.Sum(nil)
		Sm3Sum(msg)
	}
}
func TestNew(t *testing.T) {
	hw := New()
	s := hw.Sum(nil)
	t.Log(s)
}

//200000000	         7.00 ns/op
func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New()
	}
}

//500000	      3126 ns/op
func BenchmarkSum(b *testing.B) {
	buf := []byte("演示NewKeccak256()散列函数生成Hash，它的outputLen位数为32位,但它生成的Hash为64位")
	for i := 0; i < b.N; i++ {
		keccak256 := New()
		keccak256.Write(buf)
		md1 := keccak256.Sum(nil)
		b.Log(hex.EncodeToString(md1))
	}
}
