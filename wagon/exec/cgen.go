package exec

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"runtime/debug"

	"github.com/sea-project/sea-pkg/wagon/exec/internal/compile"
	"github.com/sea-project/sea-pkg/wagon/wasm"
	ops "github.com/sea-project/sea-pkg/wagon/wasm/operators"
)

const (
	FUNCTION_PREFIX = "wfun_"
	LOCAL_PREFIX    = "lc"
	VARIABLE_PREFIX = "v"
	LABEL_PREFIX    = "L_"
)

var (
	log wasm.Logger
)

// SetCGenLogger --
func SetCGenLogger(l wasm.Logger) {
	log = l
}

// CGenContext --
type CGenContext struct {
	vm            *VM
	names         []string
	mainIndex     int
	mainName      string
	keepCSource   bool
	disableGas    bool
	enableComment bool

	f            compiledFunction
	fsig         *wasm.FunctionSig
	id           uint64
	insMetas     []compile.InstructionMetadata
	branchTables []*compile.BranchTable
	labelTables  map[int]compile.Label
	labelStacks  map[int][]int

	pc      int
	opCount int
	varn    int
	stack   []int
	calln   int

	buf  *bytes.Buffer
	tabs int
}

// NewCGenContext --
func NewCGenContext(vm *VM, keepSource bool) *CGenContext {
	g := CGenContext{
		vm:          vm,
		mainIndex:   -1,
		mainName:    "thunderchain_main",
		labelStacks: make(map[int][]int),
		keepCSource: keepSource,

		stack: make([]int, 0, 1024),
		buf:   bytes.NewBuffer(nil),
	}

	return &g
}

// DisableGas --
func (g *CGenContext) DisableGas(s bool) {
	g.disableGas = s
}

// EnableComment --
func (g *CGenContext) EnableComment(s bool) {
	g.enableComment = s
}

func (g *CGenContext) resetF(f compiledFunction, id uint64) {
	g.f = f
	g.id = id
	g.insMetas = f.codeMeta.Instructions
	g.branchTables = f.codeMeta.BranchTables
	g.labelTables = f.codeMeta.LabelTables
	g.labelStacks = make(map[int][]int)

	g.pc = 0
	g.opCount = 0
	g.varn = 0
	// g.stack = g.stack[:0]
	g.stack = make([]int, 0, f.maxDepth)
	g.calln = 0

	g.buf.Reset()
	g.tabs = 0

	g.fsig = g.vm.module.FunctionIndexSpace[id].Sig
}

func (g *CGenContext) putTabs() {
	for i := 0; i < g.tabs; i++ {
		g.buf.WriteString("\t")
	}
}

func (g *CGenContext) sprintf(format string, args ...interface{}) {
	g.putTabs()
	g.buf.WriteString(fmt.Sprintf(format, args...))
}

func (g *CGenContext) writes(s string) {
	g.putTabs()
	g.buf.WriteString(s)
}

func (g *CGenContext) writeln(s string) {
	g.putTabs()
	g.buf.WriteString(s)
	g.buf.WriteString("\n")
}

func (g *CGenContext) cbytes() []byte {
	b := g.buf.Bytes()
	return b
}

func (g *CGenContext) pushStack(x int) {
	g.stack = append(g.stack, x)
	if x >= g.varn {
		g.sprintf("value_t %s%d; %s%d.vu64 = 0;\n", VARIABLE_PREFIX, x, VARIABLE_PREFIX, x)
		g.varn++
	}
}

func (g *CGenContext) popStack() int {
	x := g.topStack()
	g.stack = g.stack[:len(g.stack)-1]
	return x
}

func (g *CGenContext) topStack() int {
	return g.stack[len(g.stack)-1]
}

func (g *CGenContext) discardStack(n int) {
	g.stack = g.stack[:len(g.stack)-n]
}

func (g *CGenContext) lenStack() int {
	return len(g.stack)
}

func (g *CGenContext) isEnd() bool {
	return g.pc == len(g.f.code)
}

func (g *CGenContext) op() byte {
	ins := g.f.code[g.pc]
	if label, ok := g.labelTables[g.pc]; ok {
		// write label
		flag := false
		if g.tabs > 0 {
			g.tabs--
			flag = true
		}
		g.sprintf("\n%s%d:\n", LABEL_PREFIX, label.Index)
		if flag {
			g.tabs++
		}
		if g.disableGas {
			g.writeln("_dummy++;")
		} else {
			if ins == ops.Call {
				g.writeln("vm->gas_used += 0;")
			}
		}

		// change stack
		if tmpStack, ok := g.labelStacks[g.pc]; ok {
			log.Printf("change stack: pc:%d, new_stack:%v, old_stack:%v", g.pc, tmpStack, g.stack)
			g.stack = tmpStack
		} else {
			log.Printf("No change stack: pc:%d", g.pc)
		}
	}

	g.pc++
	if ins != ops.Call {
		var err error
		cost := GasQuickStep

		switch ins {
		case ops.Return:
			cost = 0
		case compile.OpJmp, compile.OpJmpZ, compile.OpJmpNz, ops.BrTable, compile.OpDiscard, compile.OpDiscardPreserveTop, ops.WagonNativeExec:
			cost = GasQuickStep
		default:
			gasCost := g.vm.opSet[ins].gasCost
			if gasCost == nil {
				// panic(fmt.Sprintf("gasCost nil: op:0x%x %s", ins, ops.OpSignature(ins)))
				log.Printf("gasCost nil: op:0x%x %s", ins, ops.OpSignature(ins))
				g.sprintf("panic(vm, \"[vm] operation(%s) Forbiden!!\");\n", ops.OpSignature(ins))
			} else {
				cost, err = gasCost(g.vm)
				if err != nil {
					cost = GasQuickStep
					panic(fmt.Sprintf("gasCost fail: op:0x%x %s", ins, ops.OpSignature(ins)))
				}
			}
		}
		g.genGasChecker(ins, cost)
	}

	g.opCount++
	return ins
}

func (g *CGenContext) genGasChecker(op byte, cost uint64) {
	if g.enableComment {
		g.writeln(fmt.Sprintf("// %d:%d, %s", g.pc, g.opCount, ops.OpSignature(op)))

		// @Todo: for debug
		// g.writeln(fmt.Sprintf("printf(\"pc:%d:0x%x, op:%s, gas:%d\\n\");", g.pc, op, ops.OpSignature(op), cost))
	}

	if !g.disableGas {
		g.writeln(fmt.Sprintf("if (likely(vm->gas >= %d)) {vm->gas -= %d; vm->gas_used += %d;} else {panic(vm, \"OutOfGas\");}", cost, cost, cost))
	}
}

func (g *CGenContext) fetchUint32() uint32 {
	v := endianess.Uint32(g.f.code[g.pc:])
	g.pc += 4
	return v
}

func (g *CGenContext) fetchUint64() uint64 {
	v := endianess.Uint64(g.f.code[g.pc:])
	g.pc += 8
	return v
}

func (g *CGenContext) fetchInt64() int64 {
	return int64(g.fetchUint64())
}

func (g *CGenContext) fetchBool() bool {
	return g.fetchInt8() != 0
}

func (g *CGenContext) fetchInt8() int8 {
	i := int8(g.f.code[g.pc])
	g.pc++
	return i
}

func (g *CGenContext) fetchFloat32() float32 {
	return math.Float32frombits(g.fetchUint32())
}

func (g *CGenContext) fetchFloat64() float64 {
	return math.Float64frombits(g.fetchUint64())
}

var (
	cbasic = `
// Auto Generate. Do Not Edit.

#include <stdint.h>
#include <string.h>
#include <stdlib.h>
#include <stdio.h>

typedef struct {
	void *ctx;
	uint64_t gas;
	uint64_t gas_used;
	int32_t pages;
	uint8_t *mem;

	// internal temp member
	void *_ff;
	uint32_t _findex;
} vm_t;

extern uint64_t GoFunc(vm_t*, const char*, int32_t, uint64_t*);
extern void GoPanic(vm_t*, const char*);
extern void GoRevert(vm_t*, const char*);
extern void GoExit(vm_t*, int32_t);
extern void GoGrowMemory(vm_t*, int32_t);

static inline void panic(vm_t *vm, const char *msg) {
	GoPanic(vm, msg);
}

typedef union value {
	uint64_t 	vu64;
	int64_t 	vi64;
	uint32_t 	vu32;
	int32_t 	vi32;
	uint16_t 	vu16;
	int16_t 	vi16;
	uint8_t 	vu8;
	int8_t 		vi8;
	float   	vf32;
	double 		vf64;
} value_t;

#define likely(x)       __builtin_expect((x),1)
#define unlikely(x)     __builtin_expect((x),0)

static inline uint64_t clz32(uint32_t x) {
	return __builtin_clz(x);
}
static inline uint64_t ctz32(uint32_t x) {
	return __builtin_ctz(x);
}
static inline uint64_t clz64(uint64_t x) {
	return __builtin_clzll(x);
}
static inline uint64_t ctz64(uint64_t x) {
	return __builtin_ctzll(x);
}
static inline uint64_t rotl32(uint32_t x, uint32_t r) {
	return (x << r) | (x >> (32 - r % 32));
}
static inline uint64_t rotl64(uint64_t x, uint64_t r) {
	return (x << r) | (x >> (64 - r % 64));
}
static inline uint64_t rotr32(uint32_t x, uint32_t r) {
	return (x >> r) | (x << (32 - r % 32));
}
static inline uint64_t rotr64(uint64_t x, uint64_t r) {
	return (x >> r) | (x << (64 - r % 64));
}
static inline uint32_t popcnt32(uint32_t x) {
	return (uint32_t)(__builtin_popcountl(x));
}
static inline uint32_t popcnt64(uint64_t x) {
	return (uint32_t)(__builtin_popcountll(x));
}

// ----------------

static inline uint8_t loadU8(uint8_t *p) {
	return p[0]; 
}
static inline uint16_t loadU16(uint8_t *p) {
	return ( ((uint16_t)p[0]) | (((uint16_t)p[1])<<8) );
}
static inline uint32_t loadU32(uint8_t *p) {
	return ( ((uint32_t)p[0]) | (((uint32_t)p[1])<<8) | (((uint32_t)p[2])<<16) | (((uint32_t)p[3])<<24) );
}
static inline uint64_t loadU64(uint8_t *p) {
	return ( ((uint64_t)p[0]) | (((uint64_t)p[1])<<8) | (((uint64_t)p[2])<<16) | (((uint64_t)p[3])<<24) | \
		(((uint64_t)p[4])<<32) | (((uint64_t)p[5])<<40) | (((uint64_t)p[6])<<48) | (((uint64_t)p[7])<<56) );
}
static inline void storeU8(uint8_t *p, uint8_t v) {
	p[0] = v;
}
static inline void storeU16(uint8_t *p, uint16_t v) {
	p[0] = ( ((uint8_t)v) & 0xff );
	p[1] = ( (uint8_t)((v>>8) & 0xff) );
}
static inline void storeU32(uint8_t *p, uint32_t v) {
	p[0] = ( ((uint8_t)v) & 0xff );
	p[1] = ( (uint8_t)((v>>8) & 0xff) );
	p[2] = ( (uint8_t)((v>>16) & 0xff) );
	p[3] = ( (uint8_t)((v>>24) & 0xff) );
}
static inline void storeU64(uint8_t *p, uint64_t v) {
	p[0] = ( ((uint8_t)v) & 0xff );
	p[1] = ( (uint8_t)((v>>8) & 0xff) );
	p[2] = ( (uint8_t)((v>>16) & 0xff) );
	p[3] = ( (uint8_t)((v>>24) & 0xff) );
	p[4] = ( (uint8_t)((v>>32) & 0xff) );
	p[5] = ( (uint8_t)((v>>40) & 0xff) );
	p[6] = ( (uint8_t)((v>>48) & 0xff) );
	p[7] = ( (uint8_t)((v>>56) & 0xff) );
}

#define I64Load(_p) (uint64_t)(loadU64((_p)))

#define I64Load8s(_p) (int64_t)((int8_t)(loadU8((_p))))
#define I64Load16s(_p) (int64_t)((int16_t)(loadU16((_p))))
#define I64Load32s(_p) (int64_t)((int32_t)(loadU32((_p))))

#define I64Load8u(_p) (uint64_t)((uint8_t)(loadU8((_p))))
#define I64Load16u(_p) (uint64_t)((uint16_t)(loadU16((_p))))
#define I64Load32u(_p) (uint64_t)((uint32_t)(loadU32((_p))))

#define I32Load(_p) (uint32_t)(loadU32((_p)))

#define I32Load8s(_p) (int32_t)((int8_t)(loadU8((_p))))
#define I32Load16s(_p) (int32_t)((int16_t)(loadU16((_p))))

#define I32Load8u(_p) (uint32_t)((uint8_t)(loadU8((_p))))
#define I32Load16u(_p) (uint32_t)((uint16_t)(loadU16((_p))))

`
	cenv = `
// -----------------------------------------------------
//  env api wrapper

#define MAX_U64 (uint64_t)(0xFFFFFFFFFFFFFFFF)
#define MAX_U32 (uint32_t)(0xFFFFFFFF)

#ifdef ENABLE_GAS

static inline uint32_t to_word_size(uint32_t n) {
	if (n > (MAX_U32 - 31))
		return ((MAX_U32 >> 5) + 1);
	return ((n + 31) >> 5);
}

#define USE_MEM_GAS_N(vm, n, step) {\
	uint64_t cost = to_word_size(n) * step + 2;\
	if (likely(vm->gas >= cost)) {\
		vm->gas -= cost;\
		vm->gas_used += cost;\
	} else {\
		panic(vm, "OutOfGas");\
	}\
}

#define USE_SIM_GAS_N(vm, n) {\
	uint64_t cost = n;\
	if (likely(vm->gas >= cost)) {\
		vm->gas -= cost;\
		vm->gas_used += cost;\
	} else {\
		panic(vm, "OutOfGas");\
	}\
}

#else
#define USE_MEM_GAS_N(vm, n, step) 
#define USE_SIM_GAS_N(vm, n) 
#endif

static inline uint32_t TCMemcpy(vm_t *vm, uint32_t dst, uint32_t src, uint32_t n) {
	USE_MEM_GAS_N(vm, n, 3)
	memcpy(vm->mem+dst, vm->mem+src, n);
	return dst;
}

static inline uint32_t TCMemset(vm_t *vm, uint32_t src, int c, uint32_t n) {
	USE_MEM_GAS_N(vm, n, 3)
	memset(vm->mem+src, c, n);
	return src;
}

static inline uint32_t TCMemmove(vm_t *vm, uint32_t dst, uint32_t src, uint32_t n) {
	USE_MEM_GAS_N(vm, n, 3)
	memmove(vm->mem+dst, vm->mem+src, n);
	return dst;
}

static inline int TCMemcmp(vm_t *vm, uint32_t s1, uint32_t s2, uint32_t n) {
	USE_MEM_GAS_N(vm, n, 1)
	return memcmp(vm->mem+s1, vm->mem+s2, n);
}

static inline int TCStrcmp(vm_t *vm, uint32_t s1, uint32_t s2) {
#ifdef ENABLE_GAS
	uint32_t n1 = strlen((const char *)(vm->mem+s1));
	uint32_t n2 = strlen((const char *)(vm->mem+s2));
	uint32_t n = (n1 > n2) ? n2 : n1;
	USE_MEM_GAS_N(vm, n, 1)
#endif
	return strcmp((const char *)(vm->mem+s1), (const char *)(vm->mem+s2));
}

static inline uint32_t TCStrcpy(vm_t *vm, uint32_t dst, uint32_t src) {
#ifdef ENABLE_GAS
	uint32_t n = strlen((const char *)(vm->mem+src));
	USE_MEM_GAS_N(vm, n, 3)
#endif
	strcpy((char *)(vm->mem+dst), (const char *)(vm->mem+src));
	return dst;
}

static inline uint32_t TCStrlen(vm_t *vm, uint32_t s) {
	USE_SIM_GAS_N(vm, 2)
	return strlen((const char *)(vm->mem + s));
}

static inline int TCAtoi(vm_t *vm, uint32_t s) {
	USE_SIM_GAS_N(vm, 20)
	return atoi((const char *)(vm->mem+s));
}

static inline int64_t TCAtoi64(vm_t *vm, uint32_t s) {
	USE_SIM_GAS_N(vm, 20)
	return atoll((const char *)(vm->mem + s));
}

static inline void TCRequire(vm_t *vm, int32_t cond) {
	USE_SIM_GAS_N(vm, 2)
	if (cond == 0) {
		GoRevert(vm, "TCRequire");
	}
}

static inline void TCRequireWithMsg(vm_t *vm, int32_t cond, uint32_t msg) {
#ifdef ENABLE_GAS
	uint32_t n = strlen((const char *)(vm->mem+msg));
	USE_MEM_GAS_N(vm, n, 1)
#endif
	if (cond == 0) {
		GoRevert(vm, (const char *)(vm->mem+msg));
	}
}

static inline void TCAssert(vm_t *vm, int32_t cond) {
	USE_SIM_GAS_N(vm, 2)
	if (cond == 0) {
		GoRevert(vm, "TCAssert");
	}
}

static inline void TCRevert(vm_t *vm) {
	USE_SIM_GAS_N(vm, 2)
	GoRevert(vm, "TCRevert");
}

static inline void TCRevertWithMsg(vm_t *vm, uint32_t msg) {
#ifdef ENABLE_GAS
	uint32_t n = strlen((const char *)(vm->mem+msg));
	USE_MEM_GAS_N(vm, n, 1)
#endif
	GoRevert(vm, (const char *)(vm->mem+msg));
}

static inline void TCAbort(vm_t *vm) {
	USE_SIM_GAS_N(vm, 2)
	panic(vm, "Abort");
}

static inline void TCExit(vm_t *vm, int32_t n) {
	USE_SIM_GAS_N(vm, 2)
	GoExit(vm, n);
}

`
)

// Compile --
func (g *CGenContext) Compile(code []byte, path, name string) (string, error) {
	os.MkdirAll(path, os.ModeDir)
	in := fmt.Sprintf("%s/%s.c", path, name)
	out := fmt.Sprintf("%s/%s.so", path, name)

	if err := ioutil.WriteFile(in, code, 0644); err != nil {
		log.Printf("WriteFile %s fail: %s", in, err)
		return "", err
	}

	if !g.keepCSource {
		defer func() {
			os.Remove(in)
		}()
	}

	cmd := exec.Command("gcc", "-fPIC", "-O2", "-shared", "-o", out, in)
	cmdOut, err := cmd.CombinedOutput()
	log.Printf("compiler output: %s", string(cmdOut))
	return out, err
}

// Generate --
func (g *CGenContext) Generate() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// header
	buf.WriteString(cbasic)
	if !g.disableGas {
		buf.WriteString("\n#define ENABLE_GAS\n\n")
	}
	buf.WriteString(cenv)
	buf.WriteString("\n//--------------------------\n\n")

	for index, f := range g.vm.funcs {
		name := g.vm.module.FunctionIndexSpace[index]
		if _, ok := f.(goFunction); ok {
			log.Printf("[Generate] goFunction: index:%d, name:%s", index, name)
		} else {
			log.Printf("[Generate] localFunction: index:%d, name:%s", index, name)
		}
	}

	if g.vm.module.Import != nil {
		for index, entry := range g.vm.module.Import.Entries {
			log.Printf("[Generate] Import: index:%d, entry:%s", index, entry)
		}
	}

	if g.vm.module.Export != nil {
		for name, entry := range g.vm.module.Export.Entries {
			log.Printf("[Generate] Export: name:%s, entry:%s", name, entry.String())
		}
	}

	// function declation
	names := make([]string, 0, len(g.vm.funcs))
	module := g.vm.module
	for index, f := range g.vm.funcs {
		if _, ok := f.(goFunction); ok {
			name := module.FunctionIndexSpace[index].Name
			if name == "" {
				log.Printf("[Generate] goFunction without name: func_index:%d", index)
				return buf.Bytes(), fmt.Errorf("goFunction without name")
			}
			names = append(names, name)
			continue
		}

		entry := module.FunctionIndexSpace[index]
		if entry.Name == g.mainName {
			g.mainIndex = index
			log.Printf("skip thunderchain_main: index=%d", index)
			continue
		}
		if entry.Name != "" {
			log.Printf("[Generate] declation: %s", module.FunctionIndexSpace[index].Name)
		}

		fsig := entry.Sig
		buf.WriteString(fmt.Sprintf("static %s %s%d(vm_t*", fsigReturnCType(fsig), FUNCTION_PREFIX, index))
		for _, argType := range fsig.ParamTypes {
			buf.WriteString(fmt.Sprintf(", %s", valueTypeToCType(argType)))
		}
		buf.WriteString(");\n")
	}

	g.names = names
	// static const char *env_func_names[] = {"", ""};
	buf.WriteString("\nstatic const char *env_func_names[] = {")
	for index, name := range names {
		if index > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("\"%s\"", name))
	}
	buf.WriteString("};\n")
	buf.WriteString("\n//--------------------------\n\n")
	log.Printf("env names: %v", names)

	// static uint64_t globals[] = {};
	buf.WriteString("\nstatic uint64_t globals[] = {")
	for i, global := range module.GlobalIndexSpace {
		val, err := module.ExecInitExpr(global.Init)
		if err != nil {
			log.Printf("[Generate]: module.ExecInitExpr fail: %s", err)
			return buf.Bytes(), err
		}

		if i > 0 {
			buf.WriteString(", ")
		}
		switch v := val.(type) {
		case int32, int64:
			buf.WriteString(fmt.Sprintf("0x%x", v))
		default:
			log.Printf("[Generate]: invalid global type")
			panic("")
		}
	}
	buf.WriteString("};\n")

	// static uint32_t table_index_space[] = {}
	buf.WriteString("\nstatic uint32_t table_index_space[] = {")
	for i, val := range module.TableIndexSpace[0] {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%d", val))
	}
	buf.WriteString("};\n")

	// static const uint32_t total_funcs_cnt = 100;
	buf.WriteString(fmt.Sprintf("\nstatic const uint32_t total_funcs_cnt = %d;\n", len(g.vm.funcs)))

	// static void* funcs_addr_table[] = {xxx, xxx};
	buf.WriteString("\nstatic void* funcs_addr_table[] = {")
	for index, f := range g.vm.funcs {
		if index > 0 {
			buf.WriteString(", ")
		}

		if index == g.mainIndex {
			buf.WriteString("NULL")
			continue
		}

		if _, ok := f.(compiledFunction); ok {
			buf.WriteString(fmt.Sprintf("%s%d", FUNCTION_PREFIX, index))
			continue
		}

		name := g.vm.module.FunctionIndexSpace[index].Name
		switch name {
		case "exit":
			buf.WriteString("TCExit")
		case "abort":
			buf.WriteString("TCAbort")
		case "memcpy":
			buf.WriteString("TCMemcpy")
		case "memset":
			buf.WriteString("TCMemset")
		case "memmove":
			buf.WriteString("TCMemmove")
		case "memcmp":
			buf.WriteString("TCMemcmp")
		case "strcmp":
			buf.WriteString("TCStrcmp")
		case "strcpy":
			buf.WriteString("TCStrcpy")
		case "strlen":
			buf.WriteString("TCStrlen")
		case "atoi":
			buf.WriteString("TCAtoi")
		case "atoi64":
			buf.WriteString("TCAtoi64")
		case "TC_Assert":
			buf.WriteString("TCAssert")
		case "TC_Require":
			buf.WriteString("TCRequire")
		case "TC_RequireWithMsg":
			buf.WriteString("TCRequireWithMsg")
		case "TC_Revert":
			buf.WriteString("TCRevert")
		case "TC_RevertWithMsg":
			buf.WriteString("TCRevertWithMsg")
		default:
			buf.WriteString("NULL")
		}
	}
	buf.WriteString("};\n")

	// ---------------------------

	// function code
	buf.WriteString("\n")
	for index, f := range g.vm.funcs {
		cf, ok := f.(compiledFunction)
		if ok {
			g.resetF(cf, uint64(index))
			code, err := g.doGenerateF()
			if err != nil {
				log.Printf("[Generate] doGenerateF %dth fail: %s", index, string(code))
				// log.Printf("buffer: %s", buf.String())
				return buf.Bytes(), err
			}
			buf.Write(code)
		}
	}

	return buf.Bytes(), nil
}

func (g *CGenContext) doGenerateF() (_ []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic: %s", string(debug.Stack()))
			log.Printf("Func %dth code: %s", g.id, g.buf.String())

			switch e := r.(type) {
			case error:
				err = e
			default:
				err = fmt.Errorf("panic: %v", e)
			}
		}
	}()

	funcName := g.mainName
	if g.id != uint64(g.mainIndex) {
		funcName = fmt.Sprintf("%s%d", FUNCTION_PREFIX, g.id)
	}

	fsig := g.fsig
	g.sprintf("%s %s(vm_t *vm", fsigReturnCType(fsig), funcName)
	for argIndex, argType := range fsig.ParamTypes {
		g.sprintf(",%s %s%d", valueTypeToCType(argType), LOCAL_PREFIX, argIndex)
	}
	g.writes(") {\n")
	g.tabs++

	// generate locals
	for i := g.f.args; i < g.f.totalLocalVars; i++ {
		g.sprintf("uint64_t %s%d = 0;\n", LOCAL_PREFIX, i)
	}
	if g.disableGas {
		g.writeln("uint8_t _dummy = 0;\n")
	}

	// @Todo: for debug
	// if g.enableComment {
	// 	if g.id == uint64(g.mainIndex) {
	// 		g.sprintf("printf(\"thunderchain_main begin\\n\");\n")
	// 	}
	// }

	// generate code body
	var op byte
	for !g.isEnd() {
		op = g.op()
		log.Printf("Generate %dth [%d:%d] op: %s", g.id, g.pc, len(g.f.code), ops.OpSignature(op))
		switch op {
		case ops.Nop:
		case ops.Drop:
			g.popStack()
		case ops.Unreachable:
			g.sprintf("panic(vm, \"Unreachable\");")
		case compile.OpJmp, compile.OpJmpNz, compile.OpJmpZ, ops.BrTable:
			genJmpOp(g, op)
		case ops.CallIndirect:
			err = genCallIndirectOp(g, op)
		case ops.Call:
			err = genCallOp(g, op)
		case compile.OpDiscard:
			n := g.fetchUint64()
			g.discardStack(int(n))
		case compile.OpDiscardPreserveTop:
			top := g.topStack()
			n := g.fetchUint64()
			g.discardStack(int(n))
			g.pushStack(top)
		case ops.Return:
			genReturnOp(g, op)
		case ops.Select:
			genSelectOp(g, op)
		case ops.CurrentMemory, ops.GrowMemory:
			genMemoryOp(g, op)
		case ops.GetLocal, ops.SetLocal, ops.TeeLocal:
			genLocalOp(g, op)
		case ops.GetGlobal, ops.SetGlobal:
			genGlobalOp(g, op)
		case ops.I32Const, ops.I64Const:
			genConstOp(g, op)
		case ops.I32Add, ops.I32Sub, ops.I32Mul, ops.I32DivU, ops.I32RemU, ops.I32DivS, ops.I32RemS,
			ops.I32And, ops.I32Or, ops.I32Xor,
			ops.I32Shl, ops.I32ShrS, ops.I32ShrU,
			ops.I32LeS, ops.I32LeU, ops.I32LtS, ops.I32LtU, ops.I32GeS, ops.I32GeU, ops.I32GtS, ops.I32GtU, ops.I32Eq, ops.I32Ne:
			genI32BinOp(g, op)
		case ops.I64Add, ops.I64Sub, ops.I64Mul, ops.I64DivS, ops.I64DivU, ops.I64RemS, ops.I64RemU,
			ops.I64And, ops.I64Or, ops.I64Xor,
			ops.I64Shl, ops.I64ShrS, ops.I64ShrU,
			ops.I64LeS, ops.I64LeU, ops.I64LtS, ops.I64LtU, ops.I64GeS, ops.I64GeU, ops.I64GtS, ops.I64GtU, ops.I64Eq, ops.I64Ne:
			genI64BinOp(g, op)
		case ops.I32Rotl, ops.I32Rotr, ops.I64Rotl, ops.I64Rotr:
			genBinFuncOp(g, op)
		case ops.I32Eqz, ops.I64Eqz:
			genEqzOp(g, op)
		case ops.I32Clz, ops.I32Ctz, ops.I64Clz, ops.I64Ctz, ops.I32Popcnt, ops.I64Popcnt:
			genUnFuncOp(g, op)
		case ops.I32WrapI64, ops.I64ExtendSI32, ops.I64ExtendUI32:
			genConvertOp(g, op)
		case ops.I64Load, ops.I64Load32s, ops.I64Load32u, ops.I64Load16s, ops.I64Load16u, ops.I64Load8s, ops.I64Load8u,
			ops.I32Load, ops.I32Load16s, ops.I32Load16u, ops.I32Load8s, ops.I32Load8u:
			genLoadOp(g, op)
		case ops.I64Store, ops.I64Store32, ops.I64Store16, ops.I64Store8,
			ops.I32Store, ops.I32Store16, ops.I32Store8:
			genStoreOp(g, op)
		case ops.F64Load, ops.F32Load, ops.F64Store, ops.F32Store, ops.F64Const, ops.F32Const,
			ops.F64Add, ops.F64Sub, ops.F64Mul, ops.F64Div, ops.F64Eq, ops.F64Ne, ops.F64Le, ops.F64Lt, ops.F64Ge, ops.F64Gt, ops.F64Min, ops.F64Max, ops.F64Copysign,
			ops.F32Add, ops.F32Sub, ops.F32Mul, ops.F32Div, ops.F32Eq, ops.F32Ge, ops.F32Ne, ops.F32Gt, ops.F32Lt, ops.F32Le, ops.F32Min, ops.F32Max, ops.F32Copysign,
			ops.F32Abs, ops.F32Neg, ops.F32Ceil, ops.F32Floor, ops.F32Trunc, ops.F32Nearest, ops.F32Sqrt,
			ops.F64Abs, ops.F64Neg, ops.F64Ceil, ops.F64Floor, ops.F64Trunc, ops.F64Nearest, ops.F64Sqrt,
			ops.I32TruncSF32, ops.I32TruncSF64, ops.I32TruncUF32, ops.I32TruncUF64,
			ops.I64TruncSF32, ops.I64TruncUF32, ops.I64TruncSF64, ops.I64TruncUF64,
			ops.I32ReinterpretF32, ops.I64ReinterpretF64, ops.F32ReinterpretI32, ops.F64ReinterpretI64,
			ops.F32ConvertSI32, ops.F32ConvertUI32, ops.F32ConvertSI64, ops.F32ConvertUI64, ops.F32DemoteF64,
			ops.F64ConvertSI32, ops.F64ConvertSI64, ops.F64ConvertUI32, ops.F64ConvertUI64, ops.F64PromoteF32:
			genFloatOp(g, op)

		default:
			err = fmt.Errorf("Not Support op(0x%x): %s", op, ops.OpSignature(op))
		}

		if err != nil {
			return g.cbytes(), err
		}
	}

	// @Todo: for debug
	// if g.enableComment {
	// 	if g.id == uint64(g.mainIndex) {
	// 		g.sprintf("printf(\"thunderchain_main end\\n\");\n")
	// 	}
	// }

	if op != ops.Return {
		genReturnOp(g, ops.Return)
	} else {
		log.Printf("last op is ops.Return")
	}
	g.tabs--
	g.writes("}\n\n")

	return g.cbytes(), nil
}

// --------------------------------------------------------

func genReturnOp(g *CGenContext, op byte) {
	var buf string
	if g.f.returns {
		if g.lenStack() > 0 {
			buf = fmt.Sprintf("return %s%d.%s;", VARIABLE_PREFIX, g.topStack(), valueTypeToUnionType(g.fsig.ReturnTypes[0]))
		} else {
			log.Printf("[genReturnOp]: lackof return value")
			buf = "return 0;"
		}
	} else {
		buf = "return;"
	}

	g.writeln(buf)
	log.Printf("[genReturnOp] op:0x%x, %s", op, buf)
}

func genLocalOp(g *CGenContext, op byte) {
	index := g.fetchUint32()
	var buf string

	switch op {
	case ops.GetLocal:
		g.pushStack(g.varn)
		buf = fmt.Sprintf("%s%d.vu64 = %s%d;", VARIABLE_PREFIX, g.topStack(), LOCAL_PREFIX, index)
	case ops.SetLocal:
		buf = fmt.Sprintf("%s%d = %s%d.vu64;", LOCAL_PREFIX, index, VARIABLE_PREFIX, g.popStack())
	case ops.TeeLocal:
		buf = fmt.Sprintf("%s%d = %s%d.vu64;", LOCAL_PREFIX, index, VARIABLE_PREFIX, g.topStack())
	}

	g.writeln(buf)
	log.Printf("[genLocalOp] op:0x%x, %s", op, buf)
}

func genGlobalOp(g *CGenContext, op byte) {
	index := g.fetchUint32()
	var buf string

	switch op {
	case ops.GetGlobal:
		g.pushStack(g.varn)
		buf = fmt.Sprintf("%s%d.vu64 = globals[%d];", VARIABLE_PREFIX, g.topStack(), index)
	case ops.SetGlobal:
		buf = fmt.Sprintf("globals[%d] = %s%d.vu64;", index, VARIABLE_PREFIX, g.popStack())
	}

	g.writeln(buf)
	log.Printf("[genGlocalOp] op:0x%x, %s", op, buf)
}

func genConstOp(g *CGenContext, op byte) {
	var buf string

	g.pushStack(g.varn)
	switch op {
	case ops.I32Const:
		val := g.fetchUint32()
		buf = fmt.Sprintf("%s%d.vu32 = (uint32_t)(0x%x);", VARIABLE_PREFIX, g.topStack(), val)
	case ops.I64Const:
		val := g.fetchUint64()
		buf = fmt.Sprintf("%s%d.vu64 = (uint64_t)(0x%x);", VARIABLE_PREFIX, g.topStack(), val)
	}

	g.writeln(buf)
	log.Printf("[genConstOp] op:0x%x, %s", op, buf)
}

func genSelectOp(g *CGenContext, op byte) {
	cond := g.popStack()
	v2 := g.popStack()
	v1 := g.popStack()
	g.pushStack(g.varn) // new v
	buf := fmt.Sprintf("%s%d = %s%d.vu32 ? %s%d : %s%d;", VARIABLE_PREFIX, g.topStack(),
		VARIABLE_PREFIX, cond,
		VARIABLE_PREFIX, v1,
		VARIABLE_PREFIX, v2)
	g.writeln(buf)

	log.Printf("[genSelectOp] op:0x%x, %s", op, buf)
}

func genEqzOp(g *CGenContext, op byte) {
	var buf string

	a := g.popStack()
	g.pushStack(g.varn)
	switch op {
	case ops.I32Eqz:
		buf = fmt.Sprintf("%s%d.vi32 = (%s%d.vu32 == 0);", VARIABLE_PREFIX, g.topStack(), VARIABLE_PREFIX, a)
	case ops.I64Eqz:
		buf = fmt.Sprintf("%s%d.vi64 = (%s%d.vu64 == 0);", VARIABLE_PREFIX, g.topStack(), VARIABLE_PREFIX, a)
	}
	g.writeln(buf)

	log.Printf("[genEqzOp] op:0x%x, %s", op, buf)
}

func genI32BinOp(g *CGenContext, op byte) {
	opStr := ""
	vtype := "vu32"

	switch op {
	case ops.I32Add:
		opStr = "+"
	case ops.I32Sub:
		opStr = "-"
	case ops.I32Mul:
		opStr = "*"
	case ops.I32DivU:
		opStr = "/"
	case ops.I32RemU:
		opStr = "%"
	case ops.I32DivS:
		opStr = "/"
		vtype = "vi32"
	case ops.I32RemS:
		opStr = "%"
		vtype = "vi32"
	case ops.I32And:
		opStr = "&"
	case ops.I32Or:
		opStr = "|"
	case ops.I32Xor:
		opStr = "^"
	case ops.I32Shl:
		opStr = "<<"
	case ops.I32ShrU:
		opStr = ">>"
	case ops.I32ShrS:
		opStr = ">>"
		vtype = "vi32"
	case ops.I32LeS:
		opStr = "<="
		vtype = "vi32"
	case ops.I32LeU:
		opStr = "<="
	case ops.I32LtS:
		opStr = "<"
		vtype = "vi32"
	case ops.I32LtU:
		opStr = "<"
	case ops.I32GeS:
		opStr = ">="
		vtype = "vi32"
	case ops.I32GeU:
		opStr = ">="
	case ops.I32GtS:
		opStr = ">"
		vtype = "vi32"
	case ops.I32GtU:
		opStr = ">"
	case ops.I32Eq:
		opStr = "=="
	case ops.I32Ne:
		opStr = "!="
	default:
		panic(fmt.Sprintf("[genI32BinOp] invalid op: 0x%x", op))
	}

	// c = a op b

	// push c: a -> b -> c
	g.pushStack(g.varn)

	c := g.popStack()
	b := g.popStack()
	a := g.popStack()

	if opStr == "/" || opStr == "%" {
		g.sprintf("if (unlikely(%s%d.%s == 0)) { panic(vm, \"DivZero\"); }\n", VARIABLE_PREFIX, b, vtype)
	}
	buf := fmt.Sprintf("%s%d.%s = (%s%d.%s %s %s%d.%s);", VARIABLE_PREFIX, c, vtype,
		VARIABLE_PREFIX, a, vtype,
		opStr,
		VARIABLE_PREFIX, b, vtype)
	g.writeln(buf)
	g.pushStack(c)

	log.Printf("[genI32BinOp] op:0x%x, %s", op, buf)
}

func genI64BinOp(g *CGenContext, op byte) {
	opStr := ""
	vtype := "vu64"

	switch op {
	case ops.I64Add:
		opStr = "+"
	case ops.I64Sub:
		opStr = "-"
	case ops.I64Mul:
		opStr = "*"
	case ops.I64DivU:
		opStr = "/"
	case ops.I64RemU:
		opStr = "%"
	case ops.I64DivS:
		opStr = "/"
		vtype = "vi64"
	case ops.I64RemS:
		opStr = "%"
		vtype = "vi64"
	case ops.I64And:
		opStr = "&"
	case ops.I64Or:
		opStr = "|"
	case ops.I64Xor:
		opStr = "^"
	case ops.I64Shl:
		opStr = "<<"
	case ops.I64ShrU:
		opStr = ">>"
	case ops.I64ShrS:
		opStr = ">>"
		vtype = "vi64"
	case ops.I64LeS:
		opStr = "<="
		vtype = "vi64"
	case ops.I64LeU:
		opStr = "<="
	case ops.I64LtS:
		opStr = "<"
		vtype = "vi64"
	case ops.I64LtU:
		opStr = "<"
	case ops.I64GeS:
		opStr = ">="
		vtype = "vi64"
	case ops.I64GeU:
		opStr = ">="
	case ops.I64GtS:
		opStr = ">"
		vtype = "vi64"
	case ops.I64GtU:
		opStr = ">"
	case ops.I64Eq:
		opStr = "=="
	case ops.I64Ne:
		opStr = "!="
	default:
		panic(fmt.Sprintf("[genI64BinOp] invalid op: 0x%x", op))
	}

	g.pushStack(g.varn)
	c := g.popStack()
	b := g.popStack()
	a := g.popStack()

	if opStr == "/" || opStr == "%" {
		g.sprintf("if (unlikely(%s%d.%s == 0)) { panic(vm, \"DivZero\"); }", VARIABLE_PREFIX, b, vtype)
	}
	buf := fmt.Sprintf("%s%d.%s = (%s%d.%s %s %s%d.%s);", VARIABLE_PREFIX, c, vtype,
		VARIABLE_PREFIX, a, vtype,
		opStr,
		VARIABLE_PREFIX, b, vtype)
	g.writeln(buf)
	g.pushStack(c)

	log.Printf("[genI64BinOp] op:0x%x, %s", op, buf)
}

func genBinFuncOp(g *CGenContext, op byte) {
	fName := ""
	vtype := "vu32"

	switch op {
	case ops.I32Rotl:
		fName = "rotl32"
	case ops.I32Rotr:
		fName = "rotr32"
	case ops.I64Rotl:
		fName = "rotl64"
		vtype = "vu64"
	case ops.I64Rotr:
		fName = "rotr64"
		vtype = "vu64"
	default:
		panic(fmt.Sprintf("[genBinFuncOp] invalid op: 0x%x", op))
	}

	// c = f(a, b);
	// push c: a -> b -> c
	g.pushStack(g.varn)
	c := g.popStack()
	b := g.popStack()
	a := g.popStack()
	buf := fmt.Sprintf("%s%d.%s = %s(%s%d.%s, %s%d.%s);", VARIABLE_PREFIX, c, vtype,
		fName,
		VARIABLE_PREFIX, a, vtype,
		VARIABLE_PREFIX, b, vtype)
	g.writeln(buf)
	g.pushStack(c)

	log.Printf("[genBinFuncOp] op:0x%x, %s", op, buf)
}

func genUnFuncOp(g *CGenContext, op byte) {
	fName := ""
	vtype := "vu32"

	switch op {
	case ops.I32Clz:
		fName = "clz32"
	case ops.I32Ctz:
		fName = "ctz32"
	case ops.I32Popcnt:
		fName = "popcnt32"
	case ops.I64Clz:
		fName = "clz64"
		vtype = "vu64"
	case ops.I64Ctz:
		fName = "ctz64"
		vtype = "vu64"
	case ops.I64Popcnt:
		fName = "popcnt64"
		vtype = "vu64"
	default:
		panic(fmt.Sprintf("[genUnFuncOp] invalid op: 0x%x", op))
	}

	g.pushStack(g.varn)
	c := g.popStack()
	a := g.popStack()
	buf := fmt.Sprintf("%s%d.%s = %s(%s%d.%s);", VARIABLE_PREFIX, c, vtype,
		fName,
		VARIABLE_PREFIX, a, vtype)
	g.writeln(buf)
	g.pushStack(c)

	log.Printf("[genUnFuncOp] op:0x%x, %s", op, buf)
}

func genConvertOp(g *CGenContext, op byte) {
	dstType := ""
	srcType := ""
	_type := ""

	switch op {
	case ops.I32WrapI64:
		srcType = "vu64"
		dstType = "vu32"
		_type = "uint32_t"
	case ops.I64ExtendSI32:
		srcType = "vi32"
		dstType = "vi64"
		_type = "int64_t"
	case ops.I64ExtendUI32:
		srcType = "vu32"
		dstType = "vu64"
		_type = "uint64_t"
	default:
		panic(fmt.Sprintf("[genConvertOp] invalid op: 0x%x", op))
	}

	buf := fmt.Sprintf("%s%d.%s = (%s)(%s%d.%s);", VARIABLE_PREFIX, g.topStack(), dstType,
		_type,
		VARIABLE_PREFIX, g.topStack(), srcType)
	g.writeln(buf)

	log.Printf("[genConvertOp] op:0x%x, %s", op, buf)
}

func genLoadOp(g *CGenContext, op byte) {
	vtype := ""
	f := ""

	switch op {
	case ops.I64Load:
		vtype = "vu64"
		f = "I64Load"
	case ops.I64Load32s:
		vtype = "vi64"
		f = "I64Load32s"
	case ops.I64Load32u:
		vtype = "vu64"
		f = "I64Load32u"
	case ops.I64Load16s:
		vtype = "vi64"
		f = "I64Load16s"
	case ops.I64Load16u:
		vtype = "vu64"
		f = "I64Load16u"
	case ops.I64Load8s:
		vtype = "vi64"
		f = "I64Load8s"
	case ops.I64Load8u:
		vtype = "vu64"
		f = "I64Load8u"
	case ops.I32Load:
		vtype = "vu32"
		f = "I32Load"
	case ops.I32Load16s:
		vtype = "vi32"
		f = "I32Load16s"
	case ops.I32Load16u:
		vtype = "vu32"
		f = "I32Load16u"
	case ops.I32Load8s:
		vtype = "vi32"
		f = "I32Load8s"
	case ops.I32Load8u:
		vtype = "vu32"
		f = "I32Load8u"
	default:
		panic(fmt.Sprintf("[genLoadOp] invalid op: 0x%x", op))
	}

	g.pushStack(g.varn)

	v := g.popStack()
	offset := g.popStack()
	buf := fmt.Sprintf("%s%d.%s = %s(vm->mem + 0x%x + %s%d.vu32);", VARIABLE_PREFIX, v, vtype,
		f, g.fetchUint32(), VARIABLE_PREFIX, offset)
	g.writeln(buf)
	g.pushStack(v)

	log.Printf("[genLoadOp] op:0x%x, %s", op, buf)
}

func genStoreOp(g *CGenContext, op byte) {
	vtype := ""
	f := ""

	switch op {
	case ops.I64Store:
		vtype = "vu64"
		f = "storeU64"
	case ops.I64Store32:
		vtype = "vu32"
		f = "storeU32"
	case ops.I64Store16:
		vtype = "vu16"
		f = "storeU16"
	case ops.I64Store8:
		vtype = "vu8"
		f = "storeU8"
	case ops.I32Store:
		vtype = "vu32"
		f = "storeU32"
	case ops.I32Store16:
		vtype = "vu16"
		f = "storeU16"
	case ops.I32Store8:
		vtype = "vu8"
		f = "storeU8"
	default:
		panic(fmt.Sprintf("[genStoreOp] invalid op: 0x%x", op))
	}

	v := g.popStack()
	offset := g.popStack()
	buf := fmt.Sprintf("%s(vm->mem + 0x%x + %s%d.vu32, %s%d.%s);", f, g.fetchUint32(), VARIABLE_PREFIX, offset, VARIABLE_PREFIX, v, vtype)
	g.writeln(buf)

	log.Printf("[genStoreOp] op:0x%x, %s", op, buf)
}

func genMemoryOp(g *CGenContext, op byte) {
	var buf string

	switch op {
	case ops.CurrentMemory:
		_ = g.fetchInt8()
		g.pushStack(g.varn)
		buf = fmt.Sprintf("%s%d.vi32 = vm->pages;", VARIABLE_PREFIX, g.topStack())
	case ops.GrowMemory:
		_ = g.fetchInt8()
		n := g.popStack()
		g.pushStack(g.varn)
		buf = fmt.Sprintf("%s%d.vi32 = vm->pages; if (likely(%s%d.vi32 > vm->pages)) {GoGrowMemory(vm, %s%d.vi32);}",
			VARIABLE_PREFIX, g.topStack(), VARIABLE_PREFIX, n, VARIABLE_PREFIX, n)
	default:
		panic(fmt.Sprintf("[genMemoryOp] invalid op: 0x%x", op))
	}

	g.writeln(buf)
	log.Printf("[genMemoryOp] op:0x%x, %s", op, buf)
}

func genCallGoFunc(g *CGenContext, op byte, index uint32, fsig *wasm.FunctionSig) error {
	buf := bytes.NewBuffer(nil)

	name := g.names[index]
	log.Printf("[genCallGoFunc]: name:%s, index:%d", name, index)

	// @Todo: for debug
	// if g.enableComment {
	// 	g.sprintf(fmt.Sprintf("printf(\"call name=%s, index=%d, pc=%d\\n\");\n", name, index, g.pc))
	// }

	switch name {
	case "exit":
		buf.WriteString(fmt.Sprintf("TCExit(vm, %s%d.vi32);", VARIABLE_PREFIX, g.popStack()))
	case "abort":
		buf.WriteString("TCAbort(vm);")
	case "memcpy":
		size := g.popStack()
		src := g.popStack()
		dst := g.popStack()
		g.pushStack(g.varn)
		buf.WriteString(fmt.Sprintf("%s%d.vu32 = TCMemcpy(vm, %s%d.vu32, %s%d.vu32, %s%d.vu32);",
			VARIABLE_PREFIX, g.topStack(),
			VARIABLE_PREFIX, dst,
			VARIABLE_PREFIX, src,
			VARIABLE_PREFIX, size))
	case "memset":
		size := g.popStack()
		c := g.popStack()
		src := g.popStack()
		g.pushStack(g.varn)
		buf.WriteString(fmt.Sprintf("%s%d.vu32 = TCMemset(vm, %s%d.vu32, %s%d.vi32, %s%d.vu32);",
			VARIABLE_PREFIX, g.topStack(),
			VARIABLE_PREFIX, src,
			VARIABLE_PREFIX, c,
			VARIABLE_PREFIX, size))
	case "memmove":
		n := g.popStack()
		src := g.popStack()
		dst := g.popStack()
		g.pushStack(g.varn)
		buf.WriteString(fmt.Sprintf("%s%d.vi32 = TCMemmove(vm, %s%d.vu32, %s%d.vu32, %s%d.vu32);", VARIABLE_PREFIX, g.topStack(),
			VARIABLE_PREFIX, dst, VARIABLE_PREFIX, src, VARIABLE_PREFIX, n))
	case "memcmp":
		n := g.popStack()
		src := g.popStack()
		dst := g.popStack()
		g.pushStack(g.varn)
		buf.WriteString(fmt.Sprintf("%s%d.vi32 = TCMemcmp(vm, %s%d.vu32, %s%d.vu32, %s%d.vu32);", VARIABLE_PREFIX, g.topStack(),
			VARIABLE_PREFIX, dst, VARIABLE_PREFIX, src, VARIABLE_PREFIX, n))
	case "strcmp":
		s2 := g.popStack()
		s1 := g.popStack()
		g.pushStack(g.varn)
		buf.WriteString(fmt.Sprintf("%s%d.vi32 = TCStrcmp(vm, %s%d.vu32, %s%d.vu32);", VARIABLE_PREFIX, g.topStack(),
			VARIABLE_PREFIX, s1, VARIABLE_PREFIX, s2))
	case "strcpy":
		src := g.popStack()
		dst := g.popStack()
		g.pushStack(g.varn)
		buf.WriteString(fmt.Sprintf("%s%d.vu32 = TCStrcpy(vm, %s%d.vu32, %s%d.vu32);",
			VARIABLE_PREFIX, g.topStack(), VARIABLE_PREFIX, dst, VARIABLE_PREFIX, src))
	case "strlen":
		s := g.popStack()
		g.pushStack(g.varn)
		buf.WriteString(fmt.Sprintf("%s%d.vu32 = TCStrlen(vm, %s%d.vu32);",
			VARIABLE_PREFIX, g.topStack(), VARIABLE_PREFIX, s))
	case "atoi":
		s := g.popStack()
		g.pushStack(g.varn)
		buf.WriteString(fmt.Sprintf("%s%d.vi32 = TCAtoi(vm, %s%d.vu32);",
			VARIABLE_PREFIX, g.topStack(), VARIABLE_PREFIX, s))
	case "atoi64":
		s := g.popStack()
		g.pushStack(g.varn)
		buf.WriteString(fmt.Sprintf("%s%d.vi64 = TCAtoi64(vm, %s%d.vu32);",
			VARIABLE_PREFIX, g.topStack(), VARIABLE_PREFIX, s))
	case "TC_Assert":
		cond := g.popStack()
		buf.WriteString(fmt.Sprintf("TCAssert(vm, %s%d.vi32);", VARIABLE_PREFIX, cond))
	case "TC_Require":
		cond := g.popStack()
		buf.WriteString(fmt.Sprintf("TCRequire(vm, %s%d.vi32);", VARIABLE_PREFIX, cond))
	case "TC_RequireWithMsg":
		msg := g.popStack()
		cond := g.popStack()
		buf.WriteString(fmt.Sprintf("TCRequireWithMsg(vm, %s%d.vi32, %s%d.vu32);",
			VARIABLE_PREFIX, cond, VARIABLE_PREFIX, msg))
	case "TC_Revert":
		buf.WriteString("TCRevert(vm);")
	case "TC_RevertWithMsg":
		msg := g.popStack()
		buf.WriteString(fmt.Sprintf("TCRevertWithMsg(vm, %s%d.vu32);", VARIABLE_PREFIX, msg))
	default:
		args := make([]int, len(fsig.ParamTypes))
		for argIndex := range fsig.ParamTypes {
			args[len(fsig.ParamTypes)-argIndex-1] = g.popStack()
		}

		if len(args) > 0 {
			buf.WriteString(fmt.Sprintf("uint64_t args%d[%d] = {", g.calln, len(args)))
			for argIndex, argType := range fsig.ParamTypes {
				if argIndex > 0 {
					buf.WriteString(", ")
				}
				buf.WriteString(fmt.Sprintf("%s%d.%s", VARIABLE_PREFIX, args[argIndex], valueTypeToUnionType(argType)))
			}
			buf.WriteString("};")
			g.writeln(buf.String())
			buf.Reset()
		}

		if len(fsig.ReturnTypes) > 0 {
			g.pushStack(g.varn)
			buf.WriteString(fmt.Sprintf("%s%d.%s = GoFunc(vm, env_func_names[%d]", VARIABLE_PREFIX, g.topStack(), valueTypeToUnionType(fsig.ReturnTypes[0]), index))
		} else {
			buf.WriteString(fmt.Sprintf("GoFunc(vm, env_func_names[%d]", index))
		}

		if len(args) > 0 {
			buf.WriteString(fmt.Sprintf(", %d, &args%d[0]", len(args), g.calln))
			g.calln++
		} else {
			buf.WriteString(fmt.Sprintf(", %d, NULL", len(args)))
		}
		buf.WriteString(");")
	}

	g.writeln(buf.String())
	log.Printf("[genCallGoFunc] op:0x%x, %s", op, buf.String())
	return nil
}

func genCallOp(g *CGenContext, op byte) error {
	index := g.fetchUint32()

	module := g.vm.module
	// tIndex := module.Function.Types[index]
	// fsig := module.Types.Entries[tIndex]
	fsig := module.FunctionIndexSpace[index].Sig

	log.Printf("[genCallOp]: params:%d, stack_len:%d, func_index:%d, func_sig:%s",
		len(fsig.ParamTypes), g.lenStack(), index, fsig.String())
	if g.lenStack() < len(fsig.ParamTypes) {
		return fmt.Errorf("[genCallOp] no enough variable at stack")
	}

	if _, ok := g.vm.funcs[index].(goFunction); ok {
		return genCallGoFunc(g, op, index, fsig)
	}

	// @Todo: for debug
	// if g.enableComment {
	// 	g.sprintf(fmt.Sprintf("printf(\"call %s%d, pc=%d\\n\");\n", FUNCTION_PREFIX, index, g.pc))
	// }

	args := make([]int, len(fsig.ParamTypes))
	for argIndex := range fsig.ParamTypes {
		args[len(fsig.ParamTypes)-argIndex-1] = g.popStack()
	}

	buf := bytes.NewBuffer(nil)
	if len(fsig.ReturnTypes) > 0 {
		g.pushStack(g.varn)
		buf.WriteString(fmt.Sprintf("%s%d.%s = %s%d(vm", VARIABLE_PREFIX, g.topStack(), valueTypeToUnionType(fsig.ReturnTypes[0]), FUNCTION_PREFIX, index))
	} else {
		buf.WriteString(fmt.Sprintf("%s%d(vm", FUNCTION_PREFIX, index))
	}

	for argIndex, argType := range fsig.ParamTypes {
		buf.WriteString(fmt.Sprintf(", %s%d.%s", VARIABLE_PREFIX, args[argIndex], valueTypeToUnionType(argType)))
	}
	buf.WriteString(");")

	g.writeln(buf.String())
	log.Printf("[genCallOp] op:0x%x, %s", op, buf.String())
	return nil
}

func isStackEqual(s1, s2 []int) bool {
	if len(s1) != len(s2) {
		return false
	}

	for i, a := range s1 {
		if a != s2[i] {
			return false
		}
	}
	return true
}

func genJmpOp(g *CGenContext, op byte) {
	buf := bytes.NewBuffer(nil)

	hasStack := func(opStr string, pc int, target uint64) bool {
		oldStack := g.labelStacks[int(target)]
		if len(oldStack) > 0 {
			if !isStackEqual(oldStack, g.stack) {
				panic(fmt.Sprintf("[genJumpOp]%s label already has stack: pc:%d, target:%d", opStr, pc, target))
			}
			return true
		}
		return false
	}

	switch op {
	case compile.OpJmp:
		target := g.fetchUint64()
		if label, ok := g.labelTables[int(target)]; ok {
			buf.WriteString(fmt.Sprintf("goto %s%d;", LABEL_PREFIX, label.Index))

			if hasStack("OpJmp", g.pc, target) {
				break
			}

			log.Printf("[genJumpOp]OpJmp save stack: pc:%d, target:%d, stack[%d]:%v", g.pc, target, g.lenStack(), g.stack)
			newStack := make([]int, g.lenStack())
			copy(newStack, g.stack[0:g.lenStack()])
			g.labelStacks[int(target)] = newStack
		} else {
			log.Printf("[genJumpOp]OpJmp fail: can not find label")
			panic("")
		}
	case compile.OpJmpZ:
		target := g.fetchUint64()
		cond := g.popStack()
		if label, ok := g.labelTables[int(target)]; ok {
			buf.WriteString(fmt.Sprintf("if (likely(%s%d.vu32 == 0)) {goto %s%d;}", VARIABLE_PREFIX, cond,
				LABEL_PREFIX, label.Index))

			if hasStack("OpJmpZ", g.pc, target) {
				break
			}

			log.Printf("[genJumpOp]OpJmpZ save stack: pc:%d, target:%d, stack[%d]:%v", g.pc, target, g.lenStack(), g.stack)
			newStack := make([]int, g.lenStack())
			copy(newStack, g.stack[0:g.lenStack()])
			g.labelStacks[int(target)] = newStack
		} else {
			log.Printf("[genJumpOp]OpJmpZ fail: can not find label")
			panic("")
		}

	case compile.OpJmpNz:
		target := g.fetchUint64()
		preserveTop := g.fetchBool()
		discard := g.fetchInt64()
		cond := g.popStack()
		if label, ok := g.labelTables[int(target)]; ok {
			buf.WriteString(fmt.Sprintf("if (likely(%s%d.vu32 != 0)) {goto %s%d;}", VARIABLE_PREFIX, cond,
				LABEL_PREFIX, label.Index))

			if hasStack("OpJmpNz", g.pc, target) {
				break
			}

			newStack := make([]int, g.lenStack())
			copy(newStack, g.stack[0:g.lenStack()])

			var top int
			if preserveTop {
				top = newStack[len(newStack)-1]
			}
			newStack = newStack[:len(newStack)-int(discard)]
			if preserveTop {
				newStack = append(newStack, top)
			}

			g.labelStacks[int(target)] = newStack
			log.Printf("[genJumpOp]OpJmpNz save stack: pc:%d, target:%d, preserveTop:%v, discard:%d, g.stack:%d, stack[%d]:%v",
				g.pc, target, preserveTop, discard, g.lenStack(), len(newStack), newStack)
		} else {
			log.Printf("[genJumpOp]OpJmpNz fail: can not find label")
			panic("")
		}
	case ops.BrTable:
		index := g.fetchInt64()
		label := g.popStack()
		table := g.branchTables[index]

		buf.WriteString(fmt.Sprintf("switch(%s%d.vu32) {", VARIABLE_PREFIX, label))
		for i, target := range table.Targets {
			if target.Return {
				if !g.f.returns {
					buf.WriteString(fmt.Sprintf("case %d: return 0; ", i))
				} else {
					buf.WriteString(fmt.Sprintf("case %d: return %s%d.vu64; ", i, VARIABLE_PREFIX, g.popStack()))
				}
			} else {
				if label, ok := g.labelTables[int(target.Addr)]; ok {
					buf.WriteString(fmt.Sprintf("case %d: goto %s%d; ", i, LABEL_PREFIX, label.Index))

					if hasStack("BrTabel", g.pc, uint64(target.Addr)) {
						continue
					}

					newStack := make([]int, g.lenStack())
					copy(newStack, g.stack[0:g.lenStack()])

					var top int
					if target.PreserveTop {
						top = newStack[len(newStack)-1]
					}
					newStack = newStack[:len(newStack)-int(target.Discard)]
					if target.PreserveTop {
						newStack = append(newStack, top)
					}

					g.labelStacks[int(target.Addr)] = newStack
					log.Printf("[genJumpOp]BrTable save stack: pc:%d, i:%d, target:%d, preserveTop:%v, discard:%d, g.stack=%d, stack[%d]:%v",
						g.pc, i, target.Addr, target.PreserveTop, target.Discard, g.lenStack(), len(newStack), newStack)
				} else {
					log.Printf("[genJumpOp]BrTable fail: can not find label, i=%d", i)
					panic("")
				}
			}
		}

		target := table.DefaultTarget
		if target.Return {
			if !g.f.returns {
				buf.WriteString("default: return 0; }")
			} else {
				buf.WriteString(fmt.Sprintf("default: return %s%d.vu64; }", VARIABLE_PREFIX, g.popStack()))
			}
		} else {
			if label, ok := g.labelTables[int(target.Addr)]; ok {
				buf.WriteString(fmt.Sprintf("default: goto %s%d; }", LABEL_PREFIX, label.Index))

				if hasStack("BrTable", g.pc, uint64(target.Addr)) {
					break
				}

				newStack := make([]int, g.lenStack())
				copy(newStack, g.stack[0:g.lenStack()])

				var top int
				if target.PreserveTop {
					top = newStack[len(newStack)-1]
				}
				newStack = newStack[:len(newStack)-int(target.Discard)]
				if target.PreserveTop {
					newStack = append(newStack, top)
				}

				g.labelStacks[int(target.Addr)] = newStack
				log.Printf("[genJumpOp]BrTable save stack: pc:%d, target:%d, i:default, preserveTop:%v, discard:%d, g.stack=%d, stack[%d]:%v",
					g.pc, target.Addr, target.PreserveTop, target.Discard, g.lenStack(), len(newStack), newStack)
			} else {
				log.Printf("[genJumpOp]BrTable fail: can not find lable for DefaultTarget")
				panic("")
			}
		}

	default:
		panic(fmt.Sprintf("[genJumpOp] invalid op: 0x%x", op))
	}

	g.writeln(buf.String())
	log.Printf("[genJumpOp] op:0x%x, %s", op, buf.String())
}

func genCallIndirectOp(g *CGenContext, op byte) error {
	index := g.fetchUint32()
	fsig := g.vm.module.Types.Entries[index]
	_ = g.fetchUint32()

	tableIndex := g.popStack()

	log.Printf("[genCallIndirectOp]: params:%d, stack_len:%d, func_sig:%s",
		len(fsig.ParamTypes), g.lenStack(), fsig.String())
	if g.lenStack() < len(fsig.ParamTypes) {
		return fmt.Errorf("[genCallIndirectOp] no enough variable at stack")
	}

	args := make([]int, len(fsig.ParamTypes))
	for argIndex := range fsig.ParamTypes {
		args[len(fsig.ParamTypes)-argIndex-1] = g.popStack()
	}

	g.writeln(fmt.Sprintf("vm->_findex = table_index_space[%s%d.vu32];", VARIABLE_PREFIX, tableIndex))
	g.writeln("if (unlikely(vm->_findex >= total_funcs_cnt)) { panic(vm, \"ElemIndexOverflow\"); }")

	if len(fsig.ReturnTypes) > 0 {
		g.pushStack(g.varn)
	}

	buf := bytes.NewBuffer(nil)
	g.writeln("{")
	g.tabs++

	g.writeln("vm->_ff = funcs_addr_table[vm->_findex];")

	buf.WriteString(fmt.Sprintf("%s = (%s)(vm->_ff);", fsigToCType(&fsig, "pff"), fsigToCType(&fsig, "")))
	g.writeln(buf.String())
	log.Printf("[genCallIndirectOp]: %s", buf.String())
	buf.Reset()

	// if
	buf.WriteString(fmt.Sprintf("if (vm->_ff == NULL) { "))
	if len(args) > 0 {
		buf.WriteString(fmt.Sprintf("uint64_t args%d[%d] = {", g.calln, len(args)))
		for argIndex, argType := range fsig.ParamTypes {
			if argIndex > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprintf("%s%d.%s", VARIABLE_PREFIX, args[argIndex], valueTypeToUnionType(argType)))
		}
		buf.WriteString("}; ")
	}

	if len(fsig.ReturnTypes) > 0 {
		buf.WriteString(fmt.Sprintf("%s%d.%s = GoFunc(vm, env_func_names[vm->_findex]", VARIABLE_PREFIX, g.topStack(), valueTypeToUnionType(fsig.ReturnTypes[0])))
	} else {
		buf.WriteString(fmt.Sprintf("GoFunc(vm, env_func_names[vm->_findex]"))
	}

	if len(args) > 0 {
		buf.WriteString(fmt.Sprintf(", %d, &args%d[0]); ", len(args), g.calln))
		g.calln++
	} else {
		buf.WriteString(fmt.Sprintf(", %d, NULL); ", len(args)))
	}

	// else
	buf.WriteString("} else { ")
	if len(fsig.ReturnTypes) > 0 {
		buf.WriteString(fmt.Sprintf("%s%d.%s = ", VARIABLE_PREFIX, g.topStack(), valueTypeToUnionType(fsig.ReturnTypes[0])))
	}
	buf.WriteString("pff(vm")
	for argIndex, argType := range fsig.ParamTypes {
		buf.WriteString(fmt.Sprintf(", %s%d.%s", VARIABLE_PREFIX, args[argIndex], valueTypeToUnionType(argType)))
	}
	buf.WriteString("); }")

	g.writeln(buf.String())
	log.Printf("[genCallIndirectOp]: %s", buf.String())
	buf.Reset()

	g.tabs--
	g.writeln("}")

	return nil
}

func genFloatOp(g *CGenContext, op byte) {
	switch op {
	case ops.F64Load, ops.F32Load:
		_ = g.fetchUint32()
		// g.popStack()
		// g.pushStack(g.varn)
	case ops.F64Store, ops.F32Store:
		_ = g.popStack()
		_ = g.fetchUint32()
		g.popStack()
	case ops.F64Const:
		_ = g.fetchFloat64()
		g.pushStack(g.varn)
	case ops.F32Const:
		_ = g.fetchFloat32()
		g.pushStack(g.varn)
	case ops.F64Add, ops.F64Sub, ops.F64Mul, ops.F64Div, ops.F64Eq, ops.F64Ne, ops.F64Le, ops.F64Lt, ops.F64Ge, ops.F64Gt, ops.F64Min, ops.F64Max, ops.F64Copysign,
		ops.F32Add, ops.F32Sub, ops.F32Mul, ops.F32Div, ops.F32Eq, ops.F32Ge, ops.F32Ne, ops.F32Gt, ops.F32Lt, ops.F32Le, ops.F32Min, ops.F32Max, ops.F32Copysign:
		_ = g.popStack()
		_ = g.popStack()
		g.pushStack(g.varn)
	case ops.F32Abs, ops.F32Neg, ops.F32Ceil, ops.F32Floor, ops.F32Trunc, ops.F32Nearest, ops.F32Sqrt,
		ops.F64Abs, ops.F64Neg, ops.F64Ceil, ops.F64Floor, ops.F64Trunc, ops.F64Nearest, ops.F64Sqrt,
		ops.I32TruncSF32, ops.I32TruncSF64, ops.I32TruncUF32, ops.I32TruncUF64,
		ops.I64TruncSF32, ops.I64TruncUF32, ops.I64TruncSF64, ops.I64TruncUF64,
		ops.I32ReinterpretF32, ops.I64ReinterpretF64, ops.F32ReinterpretI32, ops.F64ReinterpretI64,
		ops.F32ConvertSI32, ops.F32ConvertUI32, ops.F32ConvertSI64, ops.F32ConvertUI64, ops.F32DemoteF64,
		ops.F64ConvertSI32, ops.F64ConvertSI64, ops.F64ConvertUI32, ops.F64ConvertUI64, ops.F64PromoteF32:

	default:
		panic(fmt.Sprintf("[genJumpOp] invalid op: 0x%x", op))
	}
}

func init() {
	log = wasm.NoopLogger{}
}

// -----------------------------------------------------

func valueTypeToCType(t wasm.ValueType) string {
	switch t {
	case wasm.ValueTypeI32:
		return "uint32_t"
	case wasm.ValueTypeI64:
		return "uint64_t"
	case wasm.ValueTypeF32:
		return "float"
	case wasm.ValueTypeF64:
		return "double"
	default:
		return "void"
	}
}

func valueTypeToUnionType(t wasm.ValueType) string {
	switch t {
	case wasm.ValueTypeI32:
		return "vu32"
	case wasm.ValueTypeI64:
		return "vu64"
	case wasm.ValueTypeF32:
		return "vf32"
	case wasm.ValueTypeF64:
		return "vf64"
	default:
		return "void"
	}
}

func fsigReturnCType(fsig *wasm.FunctionSig) string {
	if len(fsig.ReturnTypes) > 0 {
		return valueTypeToCType(fsig.ReturnTypes[0])
	}
	return "void"
}

func fsigToCType(fsig *wasm.FunctionSig, name string) string {
	buf := bytes.NewBuffer(nil)

	buf.WriteString(fmt.Sprintf("%s (*%s)(vm_t*", fsigReturnCType(fsig), name))
	for _, arg := range fsig.ParamTypes {
		buf.WriteString(fmt.Sprintf(", %s", valueTypeToCType(arg)))
	}
	buf.WriteString(")")

	return buf.String()
}
