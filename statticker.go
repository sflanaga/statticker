package statticker

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

func assert(condition bool, message string) {
	if !condition {
		panic(message)
	}
}

type StatType int

const (
	Count StatType = iota + 1
	Bytes
	Gauge // used for current value ONLY
)

type TStat struct {
	value      int64
	lastValue  int64
	firstValue int64
	name       string
	stattype   StatType
	external   func() int64
}

func Stat(name string) *TStat {
	return &TStat{0, 0, 0, name, Count, nil}
}

func (stat *TStat) statType(stattype StatType) *TStat {
	stat.stattype = stattype
	return stat
}

func (stat *TStat) setExternal(external func() int64) *TStat {
	stat.external = external
	return stat
}

func NewStat(name string, stattype StatType) *TStat {
	return &TStat{0, 0, 0, name, stattype, nil}
}

func NewStatFunc(name string, stattype StatType, get func() int64) *TStat {
	return &TStat{0, 0, 0, name, stattype, get}
}

func (stat *TStat) Add(delta int64) int64 {
	return atomic.AddInt64(&stat.value, delta)
}

type TSample struct {
	name  *string
	value int64
	delta int64
	stype StatType
}

type Ticker struct {
	msg        string
	startTime  time.Time
	lastSample time.Time
	statList   []*TStat
	interval   time.Duration
	stopper    chan bool
	ticker     *time.Ticker
	wg         sync.WaitGroup
	running    int32
	buf        []byte
	samples    []TSample
}

func NewTicker(msg string, interval time.Duration, statList []*TStat) *Ticker {
	// escape analysis I hope?!
	// C programmers dont like this pattern...
	return &Ticker{
		msg:        msg,
		startTime:  time.Now(),
		lastSample: time.Now(),
		statList:   statList,
		interval:   interval,
		stopper:    make(chan bool),
		ticker:     time.NewTicker(interval),
		running:    0,
		buf:        make([]byte, 80),
		samples:    make([]TSample, len(statList)),
	}
}

func (t *Ticker) printSample(finalOutput bool) {
	t.buf = t.buf[:0]
	now := time.Now()
	var samplePeriod time.Duration
	if !finalOutput {
		samplePeriod = now.Sub(t.lastSample)
		t.lastSample = now
	} else {
		samplePeriod = now.Sub(t.startTime)
		t.lastSample = now // silly but just in case
	}
	// stat gathering phase - should happen much faster then the print phase
	// sampleStart := time.Now()
	samples := t.samples
	for i, stat := range t.statList {
		samples[i].name = &stat.name
		samples[i].stype = stat.stattype
		if stat.external != nil {
			samples[i].value = stat.external()
		} else {
			samples[i].value = atomic.LoadInt64(&stat.value)
		}
		if !finalOutput {
			samples[i].delta = t.samples[i].value - stat.lastValue
		} else {
			samples[i].delta = t.samples[i].value - stat.firstValue
		}
		stat.lastValue = samples[i].value
	}
	// sampleEnd := time.Now()

	// now for the "slower" formatter and string stuff
	timeStr := float64(time.Since(t.startTime).Milliseconds()) / 1000.0
	if finalOutput {
		t.buf = fmt.Appendf(t.buf, "OVERALL[%s] %0.3f ", t.msg, timeStr)
	} else {
		t.buf = fmt.Appendf(t.buf, "%s %0.3f ", t.msg, timeStr)
	}
	for _, sample := range t.samples {
		var ratePerSec float64
		if !finalOutput {
			ratePerSec = float64(sample.delta) / float64(samplePeriod.Seconds())
		} else {
			ratePerSec = float64(sample.delta) / float64(samplePeriod.Seconds())
		}
		// fmt.Printf("%f  %f\n", float64(sample.delta), float64(samplePeriod.Seconds()))
		switch sample.stype {
		case Bytes:
			t.buf = fmt.Appendf(t.buf, "[%s %s/s | %s]", *sample.name, formatBytes(uint64(ratePerSec)), formatBytes(uint64(sample.value)))
		case Count:
			t.buf = fmt.Appendf(t.buf, "[%s %s/s | %s]", *sample.name, addCommas(uint64(ratePerSec)), addCommas(uint64(sample.value)))
		case Gauge:
			t.buf = fmt.Appendf(t.buf, "[%s %s]", *sample.name, addCommas(uint64(sample.value)))
		}
	}
	fmt.Fprintln(os.Stderr, string(t.buf))

	// if finalOutput {
	// 	for _, stat := range t.statList {
	// 		fmt.Printf("%s = first value: %d\n", stat.name, stat.firstValue)
	// 	}
	// }
	// ioEnd := time.Now()
	// fmt.Printf("sample time: %v, io time: %v\n", sampleEnd.Sub(sampleStart), ioEnd.Sub(sampleEnd))

}

func (t *Ticker) Start() {
	atomic.StoreInt32(&t.running, 1)
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		defer atomic.StoreInt32(&t.running, 0)

		// because we might "restart" a ticker, we need to start with new baselines for final results
		t.startTime = time.Now()
		t.lastSample = time.Now()
		for _, stat := range t.statList {
			atstart := atomic.LoadInt64(&stat.value)
			stat.lastValue = atstart
			stat.firstValue = atstart
		}

		quitValue := false
	foreverLoop:
		for !quitValue {
			select {
			case <-t.ticker.C:
				t.printSample(false)
			case quitValue = <-t.stopper:
				if quitValue {
					// redundant?
					break foreverLoop
				}
			}
		}
		println("DONE???")
		t.printSample(true)
		t.ticker.Stop()
	}()
}

func (t *Ticker) Stop() {
	assert(t.IsRunning(), "ticker is not running yet caller asked to stop it")
	if t.IsRunning() {
		t.stopper <- true
		t.wg.Wait()
	}
	assert(!t.IsRunning(), "ticker still running after stop requested")
}

func (t *Ticker) IsRunning() bool {
	return atomic.LoadInt32(&t.running) > 0
}
