package statticker

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestTicker(t *testing.T) {

	var sl []*TStat
	// getHeapMem := func() int64 {
	// 	var memStats runtime.MemStats
	// 	runtime.ReadMemStats(&memStats)
	// 	return int64(memStats.HeapAlloc)
	// }

	var count = Stat("count").statType(Count)
	var bytes = Stat("bytes").statType(Bytes)
	sl = append(sl, count)
	sl = append(sl, Stat("goroutines").statType(Gauge).setExternal(func() int64 { return int64(runtime.NumGoroutine()) }))
	// sl = append(sl, Stat("memalloc").statType(Gauge).setExternal(getHeapMem))
	sl = append(sl, bytes)

	var wg sync.WaitGroup
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// for i := 0; i < 50_000_000; i++
			for {
				count.Add(1)
				bytes.Add(100)
			}
		}()
	}

	ticker := NewTicker("my tic", time.Duration(1*time.Second), sl) // create_ticker("my ticker", time.Duration(1*1e9), sl)
	ticker.Start()
	runtime.GC()

	println("setup done")
	time.Sleep(10 * time.Second)
	// wg.Wait()
	println("main thread sleep done - stoping")
	ticker.Stop()
	// ticker.Stop()
	println("donedone")

}
