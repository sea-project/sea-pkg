package memory

import (
	"errors"
	"fmt"
)

const (
	MinAllocMemSize       = 32
	FixedStackIdx         = 16 * 1024
	MaxDataMemSize        = 16 * 1024
	DefaultMinHeapMemSize = 64 * 1024
	DefaultMaxHeapMemSize = 1024 * 1024
)

var (
	ErrMemoryNotEnough    = errors.New("memory not enough")
	ErrMemoryOutBound     = errors.New("memory out bound")
	ErrInvalidMemoryIdx   = errors.New("invalid memory idx")
	ErrInvalidParam       = errors.New("invalid param")
	ErrMemoryOverMaxLimit = errors.New("memory over max limit")
)

type MemManager struct {
	Memory        []byte
	maxHeapSize   int
	currHeapSize  int
	fixedDataIdx  int   //data section end idx
	fixedStackIdx int   //stack section end idx
	memAllocTree  []int //memory manager tree
}

var memInitPool [][]int

func init() {
	for size := DefaultMinHeapMemSize; size <= DefaultMaxHeapMemSize; size = size * 2 {
		pages := size / MinAllocMemSize
		nodeCount := 2*pages - 1
		memAllocTree := make([]int, nodeCount)
		for nodeSize, i := 2*pages, 0; i < nodeCount; i++ {
			if isPowerOf2(i + 1) {
				nodeSize /= 2
			}
			memAllocTree[i] = nodeSize
		}
		memInitPool = append(memInitPool, memAllocTree)
	}
}

func InitMemManager(dataEndIdx int, size int, maxMemSize int) (*MemManager, error) {
	if dataEndIdx-FixedStackIdx >= MaxDataMemSize {
		return nil, ErrMemoryOverMaxLimit
	}
	if !isPowerOf2(size) {
		size = fixSize(size)
	}
	if size < DefaultMinHeapMemSize {
		size = DefaultMinHeapMemSize
	}
	if maxMemSize == 0 {
		maxMemSize = DefaultMaxHeapMemSize
	}

	pages := size / MinAllocMemSize
	nodeCount := 2*pages - 1
	index := 0
	for pow := size / DefaultMinHeapMemSize; pow/2 > 0; pow = pow / 2 {
		index++
	}
	memAllocTree := make([]int, nodeCount)
	copy(memAllocTree, memInitPool[index])
	mem := make([]byte, dataEndIdx+size)

	return &MemManager{
		Memory:        mem,
		maxHeapSize:   maxMemSize,
		currHeapSize:  size,
		fixedDataIdx:  dataEndIdx,
		fixedStackIdx: FixedStackIdx,
		memAllocTree:  memAllocTree,
	}, nil
}

// HeapSize --
func (mm *MemManager) HeapSize() int {
	return mm.currHeapSize
}

func (mm *MemManager) Release() {
	// memPool.memory.Put(mm.Memory)
	// memPool.allocTree.Put(mm.memAllocTree)
}

func (mm *MemManager) CopyDataSection(data []byte) {
	copy(mm.Memory[mm.fixedStackIdx:], data)
}

func (mm *MemManager) Malloc(size int) (uint64, error) {
	if size < 0 {
		return 0, ErrInvalidParam
	}
	if size == 0 {
		return 0, nil
	}
	if !isPowerOf2(size) {
		size = fixSize(size)
	}
	pages := size / MinAllocMemSize
	if pages == 0 {
		pages = 1
	}
	if pages > mm.memAllocTree[0] {
		err := mm.GrowMem(size)
		if err != nil {
			return 0, err
		}
	}
	currPages := mm.currHeapSize / MinAllocMemSize
	index := 0
	for nodeSize := currPages; nodeSize != pages; nodeSize /= 2 {
		if left := leftChild(index); mm.memAllocTree[left] >= pages {
			index = left
		} else {
			index = rightChild(index)
		}
	}
	mm.memAllocTree[index] = 0 // mark zero as used
	offset := (index+1)*pages - currPages
	// update the parent node's size
	for index != 0 {
		index = parent(index)
		mm.memAllocTree[index] = max(mm.memAllocTree[leftChild(index)], mm.memAllocTree[rightChild(index)])
	}
	memOffset := mm.getMemOffset(offset)
	return uint64(memOffset), nil
}

func (mm *MemManager) Free(ptri64 uint64) error {
	memOffset := int(ptri64)
	if memOffset < mm.fixedDataIdx || memOffset >= len(mm.Memory) {
		return ErrMemoryOutBound
	}
	offset, err := mm.getIdxOffset(memOffset)
	if err != nil {
		return err
	}
	nodeSize := 1
	index := offset + mm.currHeapSize/MinAllocMemSize - 1
	for ; mm.memAllocTree[index] != 0; index = parent(index) {
		nodeSize *= 2
		if index == 0 {
			return ErrInvalidMemoryIdx
		}
	}
	mm.memAllocTree[index] = nodeSize
	// update parent node's size
	for index != 0 {
		index = parent(index)
		nodeSize *= 2
		leftSize := mm.memAllocTree[leftChild(index)]
		rightSize := mm.memAllocTree[rightChild(index)]
		if leftSize+rightSize == nodeSize {
			mm.memAllocTree[index] = nodeSize
		} else {
			mm.memAllocTree[index] = max(leftSize, rightSize)
		}
	}
	return nil
}

func (mm *MemManager) Realloc(ptri64 uint64, size int) (uint64, error) {
	if size < 0 {
		return 0, ErrInvalidParam
	}
	if size == 0 {
		return 0, nil
	}
	memOffset := int(ptri64)
	if memOffset == 0 {
		return mm.Malloc(size)
	}
	if memOffset < mm.fixedDataIdx || memOffset >= len(mm.Memory) {
		return 0, ErrMemoryOutBound
	}
	offset, err := mm.getIdxOffset(memOffset)
	if err != nil {
		return 0, err
	}
	nodeSize := MinAllocMemSize //real mem
	index := offset + mm.currHeapSize/MinAllocMemSize - 1
	for ; mm.memAllocTree[index] != 0; index = parent(index) {
		nodeSize *= 2
		if index == 0 {
			return 0, ErrInvalidMemoryIdx
		}
	}
	minSize := size
	if size > nodeSize {
		minSize = nodeSize
	}
	if !isPowerOf2(size) {
		size = fixSize(size)
	}
	newPtr, err := mm.Malloc(size)
	if err != nil {
		return 0, err
	}
	mm.Memcpy(newPtr, ptri64, minSize)
	mm.Free(ptri64)
	return newPtr, nil
}

func (mm *MemManager) Strlen(ptri64 uint64) (int, error) {
	ptr := int(ptri64)
	if ptr >= len(mm.Memory) {
		return 0, ErrMemoryOutBound
	}
	ptrTmp := ptr
	strLen := 0
	for ptrTmp < len(mm.Memory) && mm.Memory[ptrTmp] != byte(0) {
		strLen++
		ptrTmp++
	}
	if ptrTmp == len(mm.Memory) {
		return 0, ErrInvalidMemoryIdx
	}
	//check ptr is valid or not
	//stack section
	if ptr < mm.fixedStackIdx && ptr+strLen >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	//data section
	if ptr < mm.fixedDataIdx && ptr+strLen >= mm.fixedDataIdx {
		return 0, ErrInvalidMemoryIdx
	}
	return strLen, nil
}

func (mm *MemManager) Strcpy(desti64 uint64, srci64 uint64) (uint64, error) {
	dest := int(desti64)
	src := int(srci64)
	if dest >= len(mm.Memory) || src >= len(mm.Memory) {
		return 0, ErrMemoryOutBound
	}
	size, err := mm.Strlen(srci64)
	if err != nil {
		return 0, err
	}
	if dest+size >= len(mm.Memory) {
		return 0, ErrInvalidMemoryIdx
	}
	//not allow modify data section
	if dest < mm.fixedDataIdx && dest >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if dest < mm.fixedStackIdx && dest+size >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if dest != src {
		copy(mm.Memory[dest:], mm.Memory[src:src+size+1]) //end with \00
	}
	return desti64, nil
}

func (mm *MemManager) Strcmp(ptr1i64 uint64, ptr2i64 uint64) (int, error) {
	ptr1 := int(ptr1i64)
	ptr2 := int(ptr2i64)
	if ptr1 >= len(mm.Memory) || ptr2 >= len(mm.Memory) {
		return 0, ErrMemoryOutBound
	}
	strlen1, err1 := mm.Strlen(ptr1i64)
	strlen2, err2 := mm.Strlen(ptr2i64)
	if err1 != nil || err2 != nil {
		return 0, ErrInvalidMemoryIdx
	}
	minlen := strlen1
	if strlen1 > strlen2 {
		minlen = strlen2
	}
	for i := 0; i < minlen; i++ {
		if mm.Memory[ptr1+i] == mm.Memory[ptr2+i] {
			continue
		} else if mm.Memory[ptr1+i] < mm.Memory[ptr2+i] {
			return -1, nil
		} else {
			return 1, nil
		}
	}
	if strlen1 < strlen2 {
		return -1, nil
	}
	if strlen1 > strlen2 {
		return 1, nil
	}
	return 0, nil
}

func (mm *MemManager) GetString(ptri64 uint64) ([]byte, error) {
	ptr := int(ptri64)
	if ptr >= len(mm.Memory) {
		return nil, ErrMemoryOutBound
	}
	strlen, err := mm.Strlen(ptri64)
	if err != nil {
		return nil, ErrInvalidMemoryIdx
	}
	str := make([]byte, strlen)
	copy(str, mm.Memory[ptr:ptr+strlen])
	return str, nil
}

func (mm *MemManager) GetBytes(ptri64 uint64, size int) ([]byte, error) {
	ptr := int(ptri64)
	if ptr >= len(mm.Memory) {
		return nil, ErrMemoryOutBound
	}
	//stack section
	if ptr < mm.fixedStackIdx && ptr+size >= mm.fixedStackIdx {
		return nil, ErrInvalidMemoryIdx
	}
	//data section
	if ptr < mm.fixedDataIdx && ptr+size >= mm.fixedDataIdx {
		return nil, ErrInvalidMemoryIdx
	}
	dat := make([]byte, size)
	copy(dat, mm.Memory[ptr:ptr+size])

	return dat, nil
}

func (mm *MemManager) SetBytes(str []byte) (uint64, error) {
	strlen := len(str)
	ptri64, err := mm.Malloc(strlen + 1)
	if err != nil {
		return 0, err
	}
	ptr := int(ptri64)
	copy(mm.Memory[ptr:ptr+strlen], str)
	mm.Memory[ptr+strlen] = byte(0)
	return uint64(ptr), nil
}

func (mm *MemManager) CopyBytes(data []byte, desti64 uint64) (uint64, error) {
	dest := int(desti64)
	size := len(data)
	if dest >= len(mm.Memory) || dest+size-1 >= len(mm.Memory) {
		return 0, ErrMemoryOutBound
	}
	//not allow modify data section
	if dest < mm.fixedDataIdx && dest >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if dest < mm.fixedStackIdx && dest+size-1 >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	copy(mm.Memory[dest:dest+size], data)
	return desti64, nil
}

func (mm *MemManager) Memcpy(desti64 uint64, srci64 uint64, size int) (uint64, error) {
	dest := int(desti64)
	src := int(srci64)
	if dest >= len(mm.Memory) || dest+size-1 >= len(mm.Memory) ||
		src >= len(mm.Memory) || src+size-1 >= len(mm.Memory) {
		return 0, ErrMemoryOutBound
	}
	//not allow modify data section
	if dest < mm.fixedDataIdx && dest >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if dest < mm.fixedStackIdx && dest+size-1 >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if src < mm.fixedStackIdx && src+size-1 >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if src < mm.fixedDataIdx && src+size-1 >= mm.fixedDataIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if dest != src {
		copy(mm.Memory[dest:dest+size], mm.Memory[src:src+size])
	}
	return desti64, nil
}

func (mm *MemManager) Memset(desti64 uint64, c byte, size int) (uint64, error) {
	dest := int(desti64)
	if dest >= len(mm.Memory) || dest+size-1 >= len(mm.Memory) {
		return 0, ErrMemoryOutBound
	}
	//not allow modify data section
	if dest < mm.fixedDataIdx && dest >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if dest < mm.fixedStackIdx && dest+size-1 >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	for i := 0; i < size; i++ {
		mm.Memory[dest+i] = c
	}
	return desti64, nil
}

func (mm *MemManager) Memmove(desti64 uint64, srci64 uint64, size int) (uint64, error) {
	dest := int(desti64)
	src := int(srci64)
	if dest >= len(mm.Memory) || dest+size-1 >= len(mm.Memory) ||
		src >= len(mm.Memory) || src+size-1 >= len(mm.Memory) {
		return 0, ErrMemoryOutBound
	}
	//not allow modify data section
	if dest < mm.fixedDataIdx && dest >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if dest < mm.fixedStackIdx && dest+size-1 >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if src < mm.fixedStackIdx && src+size-1 >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if src < mm.fixedDataIdx && src+size-1 >= mm.fixedDataIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if dest != src {
		copy(mm.Memory[dest:dest+size], mm.Memory[src:src+size])
	}
	return desti64, nil
}

func (mm *MemManager) Memcmp(ptr1i64 uint64, ptr2i64 uint64, size int) (int, error) {
	ptr1 := int(ptr1i64)
	ptr2 := int(ptr2i64)
	if ptr1 >= len(mm.Memory) || ptr1+size-1 >= len(mm.Memory) ||
		ptr2 >= len(mm.Memory) || ptr2+size-1 >= len(mm.Memory) {
		return 0, ErrMemoryOutBound
	}
	if ptr1 < mm.fixedStackIdx && ptr1+size-1 >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if ptr1 < mm.fixedDataIdx && ptr1+size-1 >= mm.fixedDataIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if ptr2 < mm.fixedStackIdx && ptr2+size-1 >= mm.fixedStackIdx {
		return 0, ErrInvalidMemoryIdx
	}
	if ptr2 < mm.fixedDataIdx && ptr2+size-1 >= mm.fixedDataIdx {
		return 0, ErrInvalidMemoryIdx
	}
	for i := 0; i < size; i++ {
		if mm.Memory[ptr1+i] == mm.Memory[ptr2+i] {
			continue
		} else if mm.Memory[ptr1+i] < mm.Memory[ptr2+i] {
			return -1, nil
		} else {
			return 1, nil
		}
	}
	return 0, nil
}

func (mm *MemManager) String() string {
	return fmt.Sprintf(`
MemManager:
	maxHeapSize: %d
	currHeapSize: %d
	fixedDataIdx: %d
	fixedStackIdx: %d
	Memory: %v
	memAllocTree: %v
	`,
		mm.maxHeapSize,
		mm.currHeapSize,
		mm.fixedDataIdx,
		mm.fixedStackIdx,
		mm.Memory,
		mm.memAllocTree,
	)
}

func (mm *MemManager) getMemOffset(offset int) int {
	return mm.fixedDataIdx + offset*MinAllocMemSize
}

func (mm *MemManager) getIdxOffset(offset int) (int, error) {
	if (offset-mm.fixedDataIdx)%MinAllocMemSize != 0 {
		return 0, ErrInvalidMemoryIdx
	}
	return (offset - mm.fixedDataIdx) / MinAllocMemSize, nil
}

func (mm *MemManager) GrowMem(minSize int) error {
	if minSize <= 0 {
		return ErrInvalidParam
	}
	if mm.currHeapSize >= mm.maxHeapSize {
		return ErrMemoryOverMaxLimit
	}
	if !isPowerOf2(minSize) {
		minSize = fixSize(minSize)
	}
	heapSize := mm.currHeapSize
	sumHeapSize := mm.currHeapSize * 2
	for sumHeapSize <= mm.maxHeapSize && heapSize < minSize {
		heapSize = heapSize * 2
		sumHeapSize = sumHeapSize * 2
	}
	if sumHeapSize > mm.maxHeapSize {
		return ErrMemoryNotEnough
	}
	n := sumHeapSize / mm.currHeapSize
	preHeapSize := mm.currHeapSize / MinAllocMemSize
	nextHeapSize := preHeapSize * 2
	preMem := mm.memAllocTree
	for n > 1 {
		nextMem := make([]int, 2*nextHeapSize-1)
		preLeftIndex := preHeapSize - 1
		preSize := preHeapSize
		nextLeftIndex := nextHeapSize - 1
		for preLeftIndex >= 0 {
			copy(nextMem[nextLeftIndex:], preMem[preLeftIndex:preLeftIndex+preSize])
			preLeftIndex = parent(preLeftIndex)
			preSize = preSize / 2
			nextLeftIndex = parent(nextLeftIndex)
		}
		//initial right branch
		mm.initTree(nextMem, rightChild(0), nextHeapSize/2, nextHeapSize*2-1)
		//initial root node
		leftSize := nextMem[leftChild(0)]
		rightSize := nextMem[rightChild(0)]
		if leftSize+rightSize == nextHeapSize {
			nextMem[0] = nextHeapSize
		} else {
			nextMem[0] = max(leftSize, rightSize)
		}
		preHeapSize = nextHeapSize
		nextHeapSize = nextHeapSize * 2
		preMem = nextMem
		n = n / 2
	}

	Memory := make([]byte, mm.fixedDataIdx+sumHeapSize)
	copy(Memory, mm.Memory)
	mm.Memory = Memory
	mm.currHeapSize = sumHeapSize
	mm.memAllocTree = preMem
	return nil
}

func (mm *MemManager) initTree(tree []int, idx int, size int, length int) {
	if idx >= length {
		return
	}
	mm.initTree(tree, leftChild(idx), size/2, length)
	mm.initTree(tree, rightChild(idx), size/2, length)
	tree[idx] = size
	return
}

func isPowerOf2(size int) bool {
	return size&(size-1) == 0
}

func fixSize(size int) int {
	size |= size >> 1
	size |= size >> 2
	size |= size >> 4
	size |= size >> 8
	size |= size >> 16
	return size + 1
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func leftChild(index int) int {
	return 2*index + 1
}

func rightChild(index int) int {
	return 2*index + 2
}

func parent(index int) int {
	return (index+1)/2 - 1
}
