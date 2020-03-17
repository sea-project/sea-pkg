package wasm

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/sea-project/sea-pkg/wagon/wasm/internal/readpos"
)

var ErrInvalidMagic = errors.New("wasm: Invalid magic number")

const (
	Magic   uint32 = 0x6d736100
	Version uint32 = 0x1
)

type HostFunction interface {
	Call(index int64, ops interface{}, args []uint64) (uint64, error)
	Gas(index int64, ops interface{}, args []uint64) (uint64, error)
}

// Function represents an entry in the function index space of a module.
type Function struct {
	Sig  *FunctionSig
	Body *FunctionBody
	//Host reflect.Value
	Host HostFunction
	Name string
}

func (fct Function) String() string {
	return fmt.Sprintf("{Sig: %s, Body: %s, Host: %v, Name: %s}", fct.Sig.String(), fct.Body.String(), fct.IsHost(), fct.Name)
}

// IsHost indicates whether this function is a host function as defined in:
//  https://webassembly.github.io/spec/core/exec/modules.html#host-functions
func (fct *Function) IsHost() bool {
	//return fct.Host != reflect.Value{}
	return fct.Host != nil
}

// Module represents a parsed WebAssembly module:
// http://webassembly.org/docs/modules/
type Module struct {
	Version  uint32
	Sections []Section

	Types    *SectionTypes
	Import   *SectionImports
	Function *SectionFunctions
	Table    *SectionTables
	Memory   *SectionMemories
	Global   *SectionGlobals
	Export   *SectionExports
	Start    *SectionStartFunction
	Elements *SectionElements
	Code     *SectionCode
	Data     *SectionData
	Customs  []*SectionCustom

	// The function index space of the module (import + internal = all functions)
	FunctionIndexSpace []Function
	GlobalIndexSpace   []GlobalEntry

	// function indices into the global function space
	// the limit of each table is its capacity (cap)
	TableIndexSpace        [][]uint32
	LinearMemoryIndexSpace [][]byte

	ImportFuncMap   map[string]uint32
	ImportGlobalMap map[string]uint32

	imports struct {
		Funcs    []uint32
		Globals  int
		Tables   int
		Memories int
	}
}

func printLinearMemory(m [][]byte) string {
	buf := bytes.NewBufferString("[")
	for i, d := range m {
		buf.WriteString(fmt.Sprintf("%d(%d):%s,", i, len(d), string(d)))
	}
	buf.WriteString("]")
	return buf.String()
}

func (m *Module) String() string {
	buf := bytes.NewBufferString("{\n")
	buf.WriteString(fmt.Sprintf("SectionTypes: %s\n", m.Types.String()))
	buf.WriteString(fmt.Sprintf("SectionImports: %s\n", m.Import.String()))
	buf.WriteString(fmt.Sprintf("SectionFunctions: %s\n", m.Function.String()))
	buf.WriteString(fmt.Sprintf("SectionTables: %s\n", m.Table.String()))
	buf.WriteString(fmt.Sprintf("SectionMemory: %s\n", m.Memory.String()))
	buf.WriteString(fmt.Sprintf("SectionGlobal: %s\n", m.Global.String()))
	buf.WriteString(fmt.Sprintf("SectionExports: %s\n", m.Export.String()))
	buf.WriteString(fmt.Sprintf("SectionStart: %s\n", m.Start.String()))
	buf.WriteString(fmt.Sprintf("SectionElemetns: %s\n", m.Elements.String()))
	buf.WriteString(fmt.Sprintf("SectionCodes: %s\n", m.Code.String()))
	buf.WriteString(fmt.Sprintf("SectionDatas: %s\n", m.Data.String()))
	buf.WriteString(fmt.Sprintf("FunctionIndexSpace: %v\n", m.FunctionIndexSpace))
	buf.WriteString(fmt.Sprintf("GlobalIndexSpace: %v\n", m.GlobalIndexSpace))
	buf.WriteString(fmt.Sprintf("LinearMemoryIndexSapce: %s\n", printLinearMemory(m.LinearMemoryIndexSpace)))
	buf.WriteString(fmt.Sprintf("Imports: Funcs=%v, Globals=%d, Tables=%d, Memories=%d\n", m.imports.Funcs, m.imports.Globals, m.imports.Tables, m.imports.Memories))
	buf.WriteString(fmt.Sprintf("Self-Imports: func: %v, globals: %v\n", m.ImportFuncMap, m.ImportGlobalMap))
	buf.WriteString("}\n\n")
	return buf.String()
}

// Custom returns a custom section with a specific name, if it exists.
func (m *Module) Custom(name string) *SectionCustom {
	for _, s := range m.Customs {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// NewModule creates a new empty module
func NewModule() *Module {
	return &Module{
		Types:    &SectionTypes{},
		Import:   &SectionImports{},
		Table:    &SectionTables{},
		Memory:   &SectionMemories{},
		Global:   &SectionGlobals{},
		Export:   &SectionExports{},
		Start:    &SectionStartFunction{},
		Elements: &SectionElements{},
		Data:     &SectionData{},

		ImportFuncMap:   make(map[string]uint32),
		ImportGlobalMap: make(map[string]uint32),
	}
}

// ResolveFunc is a function that takes a module name and
// returns a valid resolved module.
type ResolveFunc func(name string) (*Module, error)

// DecodeModule is the same as ReadModule, but it only decodes the module without
// initializing the index space or resolving imports.
func DecodeModule(r io.Reader) (*Module, error) {
	reader := &readpos.ReadPos{
		R:      r,
		CurPos: 0,
	}
	m := &Module{
		ImportFuncMap:   make(map[string]uint32),
		ImportGlobalMap: make(map[string]uint32),
	}
	magic, err := readU32(reader)
	if err != nil {
		return nil, err
	}
	if magic != Magic {
		return nil, ErrInvalidMagic
	}
	if m.Version, err = readU32(reader); err != nil {
		return nil, err
	}
	if m.Version != Version {
		return nil, fmt.Errorf("wasm: unknown binary version: %d", m.Version)
	}

	err = newSectionsReader(m).readSections(reader)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// ReadModule reads a module from the reader r. resolvePath must take a string
// and a return a reader to the module pointed to by the string.
func ReadModule(r io.Reader, resolvePath ResolveFunc) (*Module, error) {
	m, err := DecodeModule(r)
	if err != nil {
		return nil, err
	}
	logger.Printf(">> Module info after DecodeModule\n%s", m.String())

	m.LinearMemoryIndexSpace = make([][]byte, 1)
	if m.Table != nil {
		m.TableIndexSpace = make([][]uint32, int(len(m.Table.Entries)))
	}

	if m.Import != nil && resolvePath != nil {
		if m.Code == nil {
			m.Code = &SectionCode{}
		}

		err := m.resolveImports(resolvePath)
		if err != nil {
			return nil, err
		}
	}

	for _, fn := range []func() error{
		m.populateGlobals,
		m.populateFunctions,
		m.populateTables,
		m.populateLinearMemory,
	} {
		if err := fn(); err != nil {
			return nil, err
		}
	}

	//logger.Printf("There are %d entries in the function index space.", len(m.FunctionIndexSpace))
	logger.Printf(">> Module info after polulate xxx\n%s", m.String())
	return m, nil
}
