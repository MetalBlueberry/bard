package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/andrepxx/go-dsp-guitar/tuner"
	"github.com/gordonklaus/portaudio"
)

func main() {

	ctx, done := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Println("done")
		done()
	}()

	portaudio.Initialize()
	defer portaudio.Terminate()
	e := newEcho(time.Second / 3)
	defer e.Close()
	chk(e.Start())
	e.AsyncFFT.Run(ctx)

	select {
	case <-time.After(30 * time.Second):
	case <-ctx.Done():
	}

	chk(e.Stop())
}

type echo struct {
	*portaudio.Stream
	buffer      []float32
	i           int
	inputDevice *portaudio.DeviceInfo
	AsyncFFT    *AsyncFFT
}

func newEcho(delay time.Duration) *echo {
	h, err := portaudio.DefaultHostApi()
	chk(err)
	var input, output *portaudio.DeviceInfo
	for _, device := range h.Devices {
		if strings.Contains(device.Name, "Jabra") {
			input = device
			output = device
			break
		}
	}
	if input == nil {
		panic("couldn't find Jabra input")
	}
	p := portaudio.LowLatencyParameters(input, output)
	p.Input.Channels = 1
	p.Output.Channels = 1
	e := &echo{
		buffer:      make([]float32, int(p.SampleRate*delay.Seconds())),
		inputDevice: input,
		AsyncFFT:    NewAsyncFFT(),
	}
	e.Stream, err = portaudio.OpenStream(p, e.processAudio)
	chk(err)
	return e
}

func (e *echo) processAudio(in, out []float32) {
	// start := time.Now()
	defer func() {
		// log.Println("took: ", time.Now().Sub(start))
	}()
	copy(out, in)
	// for i := range out {
	// 	out[i] = .7 * e.buffer[e.i]
	// 	e.buffer[e.i] = in[i]
	// 	e.i = (e.i + 1) % len(e.buffer)
	// }
	e.AsyncFFT.Process(in, e.inputDevice.DefaultSampleRate)

}

type AsyncFFT struct {
	tuner.Tuner

	buff  []float64
	rate  float64
	ready chan struct{}
	lock  sync.Mutex
}

func NewAsyncFFT() *AsyncFFT {
	return &AsyncFFT{
		Tuner: tuner.Create(),
		buff:  make([]float64, 0),
		ready: make(chan struct{}),
		lock:  sync.Mutex{},
	}

}

func (async *AsyncFFT) Process(samples []float32, rate float64) {
	async.lock.Lock()
	defer async.lock.Unlock()

	select {
	case async.ready <- struct{}{}:
		async.buff = async.buff[0:0]
		// log.Println(len(async.buff))
		async.rate = rate
		for i := range samples {
			async.buff = append(async.buff, float64(samples[i]))
		}
		// log.Println("copy: ", async.buff)
		// log.Println("real: ", samples)
	default:
	}

}

func (async *AsyncFFT) Run(ctx context.Context) {
	previous := ""
	go func() {
		for {
			select {
			case <-async.ready:

				async.lock.Lock()
				async.lock.Unlock()

				// log.Println("process: ", async.buff)
				async.Tuner.Process(async.buff, uint32(async.rate))
				result, err := async.Tuner.Analyze()
				if err != nil {
					panic(err)
				}
				if previous != result.Note() {
					log.Println(result)
					previous = result.Note()
				}
			case <-ctx.Done():
				return
			}

		}
	}()
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
