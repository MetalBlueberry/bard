// Copyright 2016 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/metalblueberry/bard/pkg/tuner"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	ctx  context.Context
	echo *audioTee
	buff []float64

	vertices []ebiten.Vertex
	indices  []uint16
}

func (g *Game) Update() error {
	return g.ctx.Err()
}

func (g *Game) Draw(screen *ebiten.Image) {

	g.buff = g.echo.AsyncFFT.CoppyBuffer(g.buff)
	r := g.echo.AsyncFFT.Result()
	ebitenutil.DebugPrint(screen, fmt.Sprintf("%#v", r.NoteValues))

	up := screen.SubImage(image.Rect(0, 0, screen.Bounds().Dx(), screen.Bounds().Dy()/2)).(*ebiten.Image)
	down := screen.SubImage(image.Rect(0, screen.Bounds().Dy()/2, screen.Bounds().Dx(), screen.Bounds().Dy())).(*ebiten.Image)
	g.drawWave(up, g.buff, 1)
	notes := make([]float64, len(r.NoteValues))
	for i, n := range r.NoteValues {
		// log.Println(n.Value)
		// log.Println(n.Value)
		if n.Value > 100 {
			notes[i] = 100
		} else {
			notes[i] = n.Value
		}
	}
	g.drawWave(down, notes, 1000)

}

var (
	whiteImage = ebiten.NewImage(3, 3)

	// whiteSubImage is an internal sub image of whiteImage.
	// Use whiteSubImage at DrawTriangles instead of whiteImage in order to avoid bleeding edges.
	whiteSubImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

func init() {
	whiteImage.Fill(color.White)
}

func (g *Game) drawWave(screen *ebiten.Image, data []float64, size float64) {
	var path vector.Path
	mid := screen.Bounds().Min.Y + screen.Bounds().Dy()/2
	width := screen.Bounds().Dx()

	path.MoveTo(0, float32(mid))

	scale := float64(mid) / size
	for i := range data {
		y := float32((data[i] * float64(scale)) + float64(mid))
		// log.Println(y)
		path.LineTo(float32(i*width)/float32(len(data)), y)
	}

	// Draw the main line in white.
	op := &vector.StrokeOptions{}
	// op.LineCap = cap
	// op.LineJoin = join
	// op.MiterLimit = miterLimit
	op.Width = float32(1)
	vs, is := path.AppendVerticesAndIndicesForStroke(g.vertices[:0], g.indices[:0], op)
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = 1
		vs[i].ColorG = 1
		vs[i].ColorB = 1
		vs[i].ColorA = 1
	}
	screen.DrawTriangles(vs, is, whiteSubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: false,
	})

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {

	ctx, done := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Println("done")
		done()
		<-time.After(5 * time.Second)
		log.Println("TIMEOUT")
		os.Exit(1)
	}()
	log.Println("init")

	portaudio.Initialize()
	defer portaudio.Terminate()
	e := newAudioTee()
	defer e.Close()
	chk(e.Start())
	e.AsyncFFT.Run(ctx)
	defer e.Stop()

	// select {
	// case <-time.After(30 * time.Second):
	// case <-ctx.Done():
	// }

	log.Println("ready")
	ebiten.SetWindowSize(screenWidth, screenHeight)

	ebiten.SetWindowTitle("Sine Wave (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{
		ctx:  ctx,
		echo: e,
		buff: make([]float64, 0),
	}); err != nil {
		log.Println(err)
	}

}

type audioTee struct {
	*portaudio.Stream
	i           int
	inputDevice *portaudio.DeviceInfo
	AsyncFFT    *AsyncFFT
}

func newAudioTee() *audioTee {
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
	p := portaudio.HighLatencyParameters(input, output)
	// p := portaudio.LowLatencyParameters(input, output)
	p.Input.Channels = 1
	p.Output.Channels = 1
	e := &audioTee{
		inputDevice: input,
		AsyncFFT:    NewAsyncFFT(),
	}
	e.Stream, err = portaudio.OpenStream(p, e.processAudio)
	chk(err)
	return e
}

func (e *audioTee) processAudio(in, out []float32) {
	copy(out, in)
	e.AsyncFFT.Process(in, e.inputDevice.DefaultSampleRate)
}

type AsyncFFT struct {
	*tuner.Tuner
	result *tuner.Result

	buff       []float64
	rate       float64
	ready      chan struct{}
	lock       sync.Mutex
	resultLock sync.Mutex
}

func NewAsyncFFT() *AsyncFFT {
	return &AsyncFFT{
		Tuner:      tuner.Create(),
		buff:       make([]float64, 0),
		ready:      make(chan struct{}),
		lock:       sync.Mutex{},
		resultLock: sync.Mutex{},
		result:     nil,
	}

}

func (async *AsyncFFT) Process(samples []float32, rate float64) {
	async.lock.Lock()
	defer async.lock.Unlock()

	select {
	case async.ready <- struct{}{}:
		async.buff = async.buff[0:0]
		async.rate = rate
		for i := range samples {
			async.buff = append(async.buff, float64(samples[i]))
		}
	default:
	}

}

func (async *AsyncFFT) CoppyBuffer(out []float64) []float64 {
	async.lock.Lock()
	defer async.lock.Unlock()
	out = out[0:0]
	out = append(out, async.buff...)
	return out
}

func (async *AsyncFFT) Result() tuner.Result {
	async.resultLock.Lock()
	defer async.resultLock.Unlock()
	return *async.result
}

func (async *AsyncFFT) Run(ctx context.Context) {

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
				if async.result == nil || async.result.Note() != result.Note() {

					async.resultLock.Lock()
					async.result = result
					async.resultLock.Unlock()
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
