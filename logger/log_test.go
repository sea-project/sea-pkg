package logger

import (
	"testing"
)

func TestLogOut(t *testing.T) {
	SetLogger("./log.json")
	Info("🔨 mined potential block", "number", "9999")
}

func BenchmarkError(b *testing.B) {
	SetLogger("./log.json")
	for i := 0; i < b.N; i++ {
		Info("🔨 mined potential block", "number", "9999")
	}
}
