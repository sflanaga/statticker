package statticker

import (
	"fmt"
	"log"
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
	Print // completely custom output
)

type Stat struct {
	value      int64
	lastValue  int64
	firstValue int64
	Name       string
	Stattype   StatType
	external   func() int64
	print      func()
}

func NewStat(name string, stattype StatType) *Stat {
	return &Stat{0, 0, 0, name, stattype, nil, nil}
}

func NewStatFunc(name string, stattype StatType, external func() int64) *Stat {
	return &Stat{0, 0, 0, name, stattype, external, nil}
}

func (stat *Stat) WithStatType(stattype StatType) *Stat {
	stat.Stattype = stattype
	return stat
}

func (stat *Stat) WithExternal(external func() int64) *Stat {
	stat.external = external
	return stat
}

func (stat *Stat) WithPrint(print func()) *Stat {
	if stat.Stattype != Print {
		log.Fatalf("created a non-Print type Stat with a Print closure")
	}
	stat.print = print
	return stat
}

func (stat *Stat) Add(delta int64) int64 {
	return atomic.AddInt64(&stat.value, delta)
}

func (stat *Stat) Get() int64 {
	return atomic.LoadInt64(&stat.value)
}

type sample struct {
	Name  *string
	Value int64
	Delta int64
	Stype StatType
	Print func()
}

type Ticker struct {
	Msg        string
	StartTime  time.Time
	LastSample time.Time
	StatList   []*Stat
	Interval   time.Duration
	stopper    chan bool
	ticker     *time.Ticker
	wg         sync.WaitGroup
	running    int32
	Buf        []byte
	Samples    []sample
	printer    func(t *Ticker, samplePeriod time.Duration, finalOutput bool)
}

func NewTicker(msg string, interval time.Duration, statList []*Stat) *Ticker {
	// escape analysis I hope?!
	// C programmers dont like this pattern...
	return &Ticker{
		Msg:        msg,
		StartTime:  time.Now(),
		LastSample: time.Now(),
		StatList:   statList,
		Interval:   interval,
		stopper:    make(chan bool),
		ticker:     time.NewTicker(interval),
		running:    0,
		Buf:        make([]byte, 80),
		Samples:    make([]sample, len(statList)),
		printer:    defaultPrinter,
	}
}

func (t *Ticker) WithPrinter(printer func(t *Ticker, samplePeriod time.Duration, finalOutput bool)) *Ticker {
	t.printer = printer
	return t
}

func defaultPrinter(t *Ticker, samplePeriod time.Duration, finalOutput bool) {
	timeStr := float64(time.Since(t.StartTime).Milliseconds()) / 1000.0
	if finalOutput {
		t.Buf = fmt.Appendf(t.Buf, "OVERALL[%s] %0.3f ", t.Msg, timeStr)
	} else {
		t.Buf = fmt.Appendf(t.Buf, "%s %0.3f ", t.Msg, timeStr)
	}
	for _, sample := range t.Samples {
		var ratePerSec float64
		if !finalOutput {
			ratePerSec = float64(sample.Delta) / float64(samplePeriod.Seconds())
		} else {
			ratePerSec = float64(sample.Delta) / float64(samplePeriod.Seconds())
		}
		// fmt.Printf("%f  %f\n", float64(sample.delta), float64(samplePeriod.Seconds()))
		switch sample.Stype {
		case Bytes:
			t.Buf = fmt.Appendf(t.Buf, " %s: %s/s, %s", *sample.Name, FormatBytes(uint64(ratePerSec)), FormatBytes(uint64(sample.Value)))
		case Count:
			t.Buf = fmt.Appendf(t.Buf, " %s: %s/s, %s", *sample.Name, AddCommas(uint64(ratePerSec)), AddCommas(uint64(sample.Value)))
		case Gauge:
			t.Buf = fmt.Appendf(t.Buf, " %s: %s", *sample.Name, AddCommas(uint64(sample.Value)))
		}
	}
	fmt.Fprintln(os.Stderr, string(t.Buf))
	for _, sample := range t.Samples {
		if sample.Stype == Print {
			sample.Print()
		}
	}
}

func (t *Ticker) takeSamples(finalOutput bool) {
	t.Buf = t.Buf[:0]
	now := time.Now()
	var samplePeriod time.Duration
	if !finalOutput {
		samplePeriod = now.Sub(t.LastSample)
		t.LastSample = now
	} else {
		samplePeriod = now.Sub(t.StartTime)
		t.LastSample = now // silly but just in case
	}
	// stat gathering phase - should happen much faster then the print phase
	// sampleStart := time.Now()
	samples := t.Samples
	for i, stat := range t.StatList {
		samples[i].Name = &stat.Name
		samples[i].Stype = stat.Stattype
		if stat.print == nil {
			if stat.external != nil {
				samples[i].Value = stat.external()
			} else {
				samples[i].Value = atomic.LoadInt64(&stat.value)
			}
			if !finalOutput {
				samples[i].Delta = t.Samples[i].Value - stat.lastValue
			} else {
				samples[i].Delta = t.Samples[i].Value - stat.firstValue
			}
			stat.lastValue = samples[i].Value
		} else {
			samples[i].Print = stat.print
		}
	}
	// sampleEnd := time.Now()

	defaultPrinter(t, samplePeriod, finalOutput)
	// // now for the "slower" formatter and string stuff
	// timeStr := float64(time.Since(t.StartTime).Milliseconds()) / 1000.0
	// if finalOutput {
	// 	t.buf = fmt.Appendf(t.buf, "OVERALL[%s] %0.3f ", t.msg, timeStr)
	// } else {
	// 	t.buf = fmt.Appendf(t.buf, "%s %0.3f ", t.msg, timeStr)
	// }
	// for _, sample := range t.samples {
	// 	var ratePerSec float64
	// 	if !finalOutput {
	// 		ratePerSec = float64(sample.delta) / float64(samplePeriod.Seconds())
	// 	} else {
	// 		ratePerSec = float64(sample.delta) / float64(samplePeriod.Seconds())
	// 	}
	// 	// fmt.Printf("%f  %f\n", float64(sample.delta), float64(samplePeriod.Seconds()))
	// 	switch sample.stype {
	// 	case Bytes:
	// 		t.buf = fmt.Appendf(t.buf, " %s: %s/s, %s", *sample.name, FormatBytes(uint64(ratePerSec)), FormatBytes(uint64(sample.value)))
	// 	case Count:
	// 		t.buf = fmt.Appendf(t.buf, " %s: %s/s, %s", *sample.name, AddCommas(uint64(ratePerSec)), AddCommas(uint64(sample.value)))
	// 	case Gauge:
	// 		t.buf = fmt.Appendf(t.buf, " %s: %s", *sample.name, AddCommas(uint64(sample.value)))
	// 	}
	// }
	// fmt.Fprintln(os.Stderr, string(t.buf))

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
		t.StartTime = time.Now()
		t.LastSample = time.Now()
		for _, stat := range t.StatList {
			atstart := atomic.LoadInt64(&stat.value)
			stat.lastValue = atstart
			stat.firstValue = atstart
		}

		quitValue := false
	foreverLoop:
		for !quitValue {
			select {
			case <-t.ticker.C:
				t.takeSamples(false)
			case quitValue = <-t.stopper:
				if quitValue {
					// redundant?
					break foreverLoop
				}
			}
		}
		t.takeSamples(true)
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
