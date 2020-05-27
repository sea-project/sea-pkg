package types

import "github.com/syndtr/goleveldb/leveldb/iterator"

const IdealBatchSize = 100 * 1024

// 定义写操作接口
type Putter interface {
	Put(key []byte, value []byte) error
	Delete(key []byte) error
}

// 定义数据库操作接口
type Database interface {
	Putter
	Path() string
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Close() error
	NewBatch() Batch
	NewIterator() iterator.Iterator
	NewIteratorWithStart(start []byte) iterator.Iterator
}

// 批量操作接口
type Batch interface {
	Putter
	Write() error
	ValueSize() int
	Reset()
	Replay(w Putter) error
}
