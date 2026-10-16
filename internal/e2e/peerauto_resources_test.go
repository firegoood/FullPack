package e2e

import (
	"os"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/v4/process"
)

func logAutoFixtureResources(t *testing.T, label string) {
	t.Helper()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	var rss uint64
	if p, err := process.NewProcess(int32(os.Getpid())); err == nil {
		if info, err := p.MemoryInfo(); err == nil {
			rss = info.RSS
		}
	}
	t.Logf("%s: RSS=%d KiB heap=%d KiB goroutines=%d GOMAXPROCS=%d; production pressure guards retained", label, rss/1024, memory.HeapAlloc/1024, runtime.NumGoroutine(), runtime.GOMAXPROCS(0))
}

func logUnlimitedFixtureResources(t *testing.T, phase string) {
	logAutoFixtureResources(t, "unlimited fixture "+phase)
}
