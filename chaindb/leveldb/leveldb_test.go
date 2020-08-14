package leveldb

import (
	"testing"
	"time"
)

var ldb = Init("../testdata")

func Test_Put(t *testing.T) {
	if err != nil {
		panic(err)
	}
	var key []byte = []byte("qiqi")
	var value []byte = []byte("KKKKKKKKKK")
	ldb.Put(key, value)
}

func Test_Path(t *testing.T) {
	if err != nil {
		panic(err)
	}
	res := ldb.Path()
	t.Log(res)
}

func Test_Get(t *testing.T) {
	if err != nil {
		panic(err)
	}
	res, _ := ldb.Get([]byte("qiqi"))
	t.Log(string(res))
}

func Test_Has(t *testing.T) {
	if err != nil {
		panic(err)
	}
	res, _ := ldb.Has([]byte("qiqi"))
	t.Log(res)
}

func Test_Del(t *testing.T) {
	if err != nil {
		panic(err)
	}
	err = ldb.Delete([]byte("qiqi"))
	t.Log(err)
}

func TestLdbBatch_Put(t *testing.T) {
	start := time.Now()

	batch := ldb.NewBatch()

	batch.Put([]byte("123456"), []byte("123456"))
	batch.Put([]byte("456789"), []byte("456789"))
	batch.Put([]byte("789789"), []byte("789789"))
	batch.Put([]byte("123123"), []byte("123123"))
	batch.Put([]byte("456456"), []byte("456456"))
	batch.Put([]byte("111111"), []byte("111111"))
	batch.Put([]byte("222222"), []byte("222222"))
	batch.Put([]byte("333333"), []byte("333333"))
	batch.Put([]byte("444444"), []byte("444444"))
	batch.Put([]byte("555555"), []byte("555555"))
	batch.Write()

	t.Logf("填充数量s：%d", batch.ValueSize())

	value, err := ldb.Get([]byte("222222"))
	if err != nil {
		panic(err)
	}
	t.Logf("测试获取值：%s", string(value))
	t.Logf("使用时间：%s", time.Since(start))
}

func BenchmarkLevelDB_Put(b *testing.B) {
	var key []byte = []byte("qiqi")
	var value []byte = []byte("KKKKKKKKKK")
	for i := 0; i < b.N; i++ {
		ldb.Put(key, value)
		//ldb.Del(key)
	}
}

func BenchmarkLdbBatch_Put(b *testing.B) {

	batch := ldb.NewBatch()

	ldb.Delete([]byte("555555"))

	for i := 0; i < b.N; i++ {
		batch.Put([]byte("555555"), []byte("555555"))
	}
	batch.Write()
	ldb.Delete([]byte("555555"))
}
