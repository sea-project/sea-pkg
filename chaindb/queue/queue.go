package queue

import "sync"

type StackPool struct {
	sync.Mutex
	Pool *Pool
}

var sp *StackPool
var sponce sync.Once

func InitStackPool() *StackPool {
	sponce.Do(func() {
		sp = NewStackPool()
	})
	return sp
}

func NewStackPool() *StackPool {
	return &StackPool{
		Pool: new(Pool),
	}
}

func (s *StackPool) Push(x interface{}) {
	s.Lock()
	s.Pool.Push(x)
	s.Unlock()
}

func (s *StackPool) Pop() interface{} {
	s.Lock()
	defer s.Unlock()
	if s.Pool.Len() > 0 {
		return s.Pool.Pop()
	}
	return nil
}

func (s *StackPool) Len() int {
	return s.Pool.Len()
}

type Pool []interface{}

func (p Pool) Len() int { return len(p) }

func (p Pool) Less(i, j int) bool { return true }

func (p Pool) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (p *Pool) Push(x interface{}) { *p = append(*p, x) }

func (p *Pool) Pop() interface{} {
	old := *p
	n := len(old)
	x := old[n-1]
	*p = old[0 : n-1]
	return x
}
