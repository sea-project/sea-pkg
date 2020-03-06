package memorydb

import (
	"testing"
)

var key = []byte("123456")
var value = []byte("456789")

func TestMemDB_PutGet(t *testing.T) {
	// 初始化数据库
	db := Init()

	// 写入读取数据
	if err := db.Put(key, value); err != nil {
		panic("存储失败：" + err.Error())
	} else {
		values, err := db.Get(key)
		if err != nil {
			panic("获取失败：" + err.Error())
		}
		t.Logf("获取成功：%s", string(values))
	}
}

func TestMemDB_Has(t *testing.T) {
	// 初始化数据库
	db := Init()

	if err := db.Put(key, value); err != nil {
		panic("存储失败：" + err.Error())
	}

	if ok, _ := db.Has(key); ok {
		t.Log("当前键值存在")
	} else {
		t.Log(ok)
		t.Log("当前键值不存在")
	}
}

func TestMemDB_Del(t *testing.T) {
	// 初始化数据库
	db := Init()

	if err := db.Put(key, value); err != nil {
		panic("存储失败：" + err.Error())
	}

	db.Del(key)

	values, err := db.Get(key)
	if err != nil {
		t.Logf("获取失败：" + err.Error())
	} else {
		t.Logf("获取成功：%s", string(values))
	}
}

func TestMemDB_GetAllKey(t *testing.T) {
	// 初始化数据库
	db := Init()

	db.Put(key, value)
	db.Put([]byte("111111"), value)

	keys := db.GetAllKey()
	for _, v := range keys {
		t.Logf(string(v))
	}
}
