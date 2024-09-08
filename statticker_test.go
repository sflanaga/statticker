package statticker

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestSomeTimes(t *testing.T) {
	s := time.Now()
	time.Sleep(1)
	fmt.Printf("delta: %v\n", time.Now().Sub(s))
	dur := time.Duration(73*time.Hour + 25 + time.Minute*30)
	fmt.Printf("delta: %v\n", dur)
}

func TestTicker(t *testing.T) {
	var sl []*Stat
	// getHeapMem := func() int64 {
	// 	var memStats runtime.MemStats
	// 	runtime.ReadMemStats(&memStats)
	// 	return int64(memStats.HeapAlloc)
	// }

	testgaugeFunctNew := NewStatFunc("goroutines", Gauge, func() int64 { return int64(runtime.NumGoroutine()) })
	testgaugeFunctNew.WithExternal(func() int64 { return int64(runtime.NumGoroutine()) })
	testgaugeFunctNew.WithStatType(Gauge)

	var count = NewStat("count", Count)
	var bytes = NewStat("bytes", Bytes)
	sl = append(sl, count)
	sl = append(sl, NewStatFunc("goroutines", Gauge, func() int64 { return int64(runtime.NumGoroutine()) }))
	// sl = append(sl, Stat("memalloc").statType(Gauge).setExternal(getHeapMem))
	sl = append(sl, bytes)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
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
	println("main thread sleep done - stopping ticker")
	ticker.Stop()
	// ticker.Stop()
	println("donedone")

	for _, stat := range sl {
		fmt.Printf("final values [%s]: value: %d first: %d  delta: %d\n", stat.Name, stat.Get(), stat.firstValue, stat.Get()-stat.firstValue)
	}

}
