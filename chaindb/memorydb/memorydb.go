package memorydb

import (
	"errors"
	"sync"
)

// 结构体
type MemDB struct {
	db   map[string][]byte
	lock sync.RWMutex
}

var mdb *MemDB
var err error
var once sync.Once

// 单例模式初始化数据库
func Init() *MemDB {
	once.Do(func() {
		mdb, err = newMemDB()
		if err != nil {
			panic(err)
		}
	})
	return mdb
}

// 初始化内存存储
func newMemDB() (*MemDB, error) {
	return &MemDB{
		db: make(map[string][]byte),
	}, nil
}

// 写方法
func (db *MemDB) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = value
	return nil
}

// 读方法
func (db *MemDB) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if value, ok := db.db[string(key)]; ok {
		return value, nil
	}

	return nil, errors.New("key is not found")
}

// 判断是否存在
func (db *MemDB) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if _, ok := db.db[string(key)]; ok {
		return true, nil
	}
	return false, nil
}

// 删除制定键值
func (db *MemDB) Del(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))

	return nil
}

// 获取所有键
func (db *MemDB) GetAllKey() [][]byte {
	db.lock.RLock()
	defer db.lock.RUnlock()

	keys := [][]byte{}
	for key := range db.db {
		keys = append(keys, []byte(key))
	}
	return keys
}

func (db *MemDB) Path() string {
	return ""
}

// 关闭数据库(因内存数据库无需此操作，只做实现chaindb.Database的表示形式)
func (db *MemDB) Close() error {
	return nil
}
