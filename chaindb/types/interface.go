package types

// 定义写操作接口
type Putter interface {
	Put(key []byte, value []byte) error
}

// 定义数据库操作接口
type Database interface {
	Putter
	Path() string
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Del(key []byte) error
	Close() error
}

// 批量操作接口
type Batch interface {
	Putter
	Save() error
	Size() int
}
