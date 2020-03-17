package disasm

type Logger interface {
	Printf(string, ...interface{})
	Println(string, ...interface{})
}

var logger Logger

func init() {
	logger = NoopLogger{}
}

func SetLogger(l Logger) {
	logger = l
}

type NoopLogger struct{}

func (l NoopLogger) Printf(fmt string, v ...interface{})  {}
func (l NoopLogger) Println(fmt string, v ...interface{}) {}

/*
import (
	"io/ioutil"
	"log"
	"os"
)

var (
	logger  *log.Logger
	logging bool
)

func SetDebugMode(l bool) {
	w := ioutil.Discard
	logging = l

	if l {
		w = os.Stderr
	}

	logger = log.New(w, "", log.Lshortfile)
	logger.SetFlags(log.Lshortfile)

}

func init() {
	SetDebugMode(false)
}
*/
