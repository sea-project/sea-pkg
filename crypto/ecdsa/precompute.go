package ecdsa

import (
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"io/ioutil"
	"strings"
)

//go:generate go run -tags gensecp256k1 genprecomps.go

// loadS256BytePoints对用于加速secp256k1曲线标量基乘法的预计算字节点进行解压缩和反序列化。之所以使用这种方法，是因为它允许编译使用更少的ram，而且比硬编码最终内存中的数据结构要快得多。
// 同时，与计算表相比，使用这种方法在init时生成内存中的数据结构非常快。
func loadS256BytePoints() error {
	// 在生成字节点时，将没有需要加载的字节点。
	bp := secp256k1BytePoints
	if len(bp) == 0 {
		return nil
	}

	//解压缩用于加速标量基乘法的预计算表。
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(bp))
	r, err := zlib.NewReader(decoder)
	if err != nil {
		return err
	}
	serialized, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	// 反序列化预先计算的字节点，并将曲线设置为它们。
	offset := 0
	var bytePoints [32][256][3]fieldVal
	for byteNum := 0; byteNum < 32; byteNum++ {
		// All points in this window.
		for i := 0; i < 256; i++ {
			px := &bytePoints[byteNum][i][0]
			py := &bytePoints[byteNum][i][1]
			pz := &bytePoints[byteNum][i][2]
			for i := 0; i < 10; i++ {
				px.n[i] = binary.LittleEndian.Uint32(serialized[offset:])
				offset += 4
			}
			for i := 0; i < 10; i++ {
				py.n[i] = binary.LittleEndian.Uint32(serialized[offset:])
				offset += 4
			}
			for i := 0; i < 10; i++ {
				pz.n[i] = binary.LittleEndian.Uint32(serialized[offset:])
				offset += 4
			}
		}
	}
	secp256k1.bytePoints = &bytePoints
	return nil
}
