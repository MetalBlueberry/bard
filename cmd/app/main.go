package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/cmplx"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/andrepxx/go-dsp-guitar/circular"
	"github.com/gordonklaus/portaudio"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/mjibson/go-dsp/dsputils"
	"github.com/mjibson/go-dsp/fft"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
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

	g.buff = g.echo.CoppyBuffer(g.buff)

	up := screen.SubImage(image.Rect(0, 0, screen.Bounds().Dx(), screen.Bounds().Dy()/2)).(*ebiten.Image)
	down := screen.SubImage(image.Rect(0, screen.Bounds().Dy()/2, screen.Bounds().Dx(), screen.Bounds().Dy())).(*ebiten.Image)
	g.drawWave(up, g.buff, 1)

	tuneNotes := generateNotes()
	notes := make([]float64, 0, len(generateNotes()))
	X := fft.FFTReal(g.buff)

	resolution := g.echo.inputDevice.DefaultSampleRate / float64(len(g.buff))

	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}
	mplusNormalFont, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    12,
		DPI:     72,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		panic(err)
	}

	var max *NoteStruct
	var maxValue float64
	for i := range tuneNotes {
		indexValue := tuneNotes[i].Frequency / resolution
		lowIndex := int(math.Floor(indexValue))
		highIndex := int(math.Ceil(indexValue))
		r := (magnitude(lowIndex, X) + magnitude(highIndex, X)) / 2
		if maxValue < r {
			max = &tuneNotes[i]
			maxValue = r
		}
	}

	for i, tuneNote := range tuneNotes {
		indexValue := tuneNote.Frequency / resolution
		lowIndex := int(math.Floor(indexValue))
		highIndex := int(math.Ceil(indexValue))

		r := (magnitude(lowIndex, X) + magnitude(highIndex, X)) / 2
		notes = append(notes, r)

		playing := 12
		if r > 3 {
			playing = 20 + 12
			fmt.Printf("%s, %.1fHz = %.1f\n", tuneNote.Name, tuneNote.Frequency, r)
		}
		c := color.Color(color.White)
		if max.Name == tuneNote.Name {
			c = color.NRGBA{
				R: 255,
				G: 0,
				B: 0,
				A: 255,
			}
		}

		text.Draw(screen, tuneNote.Name, mplusNormalFont, i*screen.Bounds().Dx()/len(tuneNotes), playing, c)
	}
	g.drawWave(down, notes, 100)

}

func magnitude(index int, X []complex128) float64 {
	r, θ := cmplx.Polar(X[index])
	θ *= 360.0 / (2 * math.Pi)
	if dsputils.Float64Equal(r, 0) {
		θ = 0 // (When the magnitude is close to 0, the angle is meaningless)
	}
	return r
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
		y := float32((-data[i] * float64(scale)) + float64(mid))
		// log.Println(y)
		path.LineTo(float32(i*width)/float32(len(data)), y)
	}

	// Draw the main line in white.
	op := &vector.StrokeOptions{}
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
	defer e.Stop()

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
	inputDevice *portaudio.DeviceInfo

	circularBuffer circular.Buffer
	lock           sync.Mutex
}

func newAudioTee() *audioTee {
	h, err := portaudio.DefaultHostApi()
	chk(err)
	var input, output *portaudio.DeviceInfo
	for _, device := range h.Devices {
		log.Println(device)
	}
	for _, device := range h.Devices {
		if strings.Contains(device.Name, "Live camera") {
			input = device
			break
		}
	}
	if input == nil {
		panic("couldn't find input")
	}
	for _, device := range h.Devices {
		if strings.Contains(device.Name, "Jabra") {
			output = device
			break
		}
	}
	if output == nil {
		panic("couldn't find input")
	}

	p := portaudio.HighLatencyParameters(input, output)
	p.Input.Channels = 1
	p.Output.Channels = 1
	e := &audioTee{
		inputDevice:    input,
		circularBuffer: circular.CreateBuffer(44100 / 25),
	}
	e.Stream, err = portaudio.OpenStream(p, e.processAudio)
	chk(err)
	return e
}

func (e *audioTee) processAudio(in, out []float32) {
	copy(out, in)

	e.lock.Lock()
	defer e.lock.Unlock()
	for i := range in {
		e.circularBuffer.Enqueue(float64(in[i]))
	}
}

func (e *audioTee) CoppyBuffer(out []float64) []float64 {
	e.lock.Lock()
	defer e.lock.Unlock()
	if len(out) != e.circularBuffer.Length() {
		out = make([]float64, e.circularBuffer.Length())
	}
	e.circularBuffer.Retrieve(out)
	return out
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

type Notes []NoteStruct

func generateNotes() Notes {

	/*
	 * Create a list of appropriate notes.
	 */
	notes := []NoteStruct{
		// {
		// 	Name:      "H1",
		// 	Frequency: 61.7354,
		// },
		// {
		// 	Name:      "C2",
		// 	Frequency: 65.4064,
		// },
		// {
		// 	Name:      "C#2",
		// 	Frequency: 69.2957,
		// },
		// {
		// 	Name:      "D2",
		// 	Frequency: 73.4162,
		// },
		// {
		// 	Name:      "D#2",
		// 	Frequency: 77.7817,
		// },
		// {
		// 	Name:      "E2",
		// 	Frequency: 82.4069,
		// },
		// {
		// 	Name:      "F2",
		// 	Frequency: 87.3071,
		// },
		// {
		// 	Name:      "F#2",
		// 	Frequency: 92.4986,
		// },
		// {
		// 	Name:      "G2",
		// 	Frequency: 97.9989,
		// },
		// {
		// 	Name:      "G#2",
		// 	Frequency: 103.8262,
		// },
		// {
		// 	Name:      "A2",
		// 	Frequency: 110.0000,
		// },
		// {
		// 	Name:      "A#2",
		// 	Frequency: 116.5409,
		// },
		// {
		// 	Name:      "H2",
		// 	Frequency: 123.4708,
		// },
		// {
		// 	Name:      "C3",
		// 	Frequency: 130.8128,
		// },
		// {
		// 	Name:      "C#3",
		// 	Frequency: 138.5913,
		// },
		// {
		// 	Name:      "D3",
		// 	Frequency: 146.8324,
		// },
		// {
		// 	Name:      "D#3",
		// 	Frequency: 155.5635,
		// },
		// {
		// 	Name:      "E3",
		// 	Frequency: 164.8138,
		// },
		// {
		// 	Name:      "F3",
		// 	Frequency: 174.6141,
		// },
		// {
		// 	Name:      "F#3",
		// 	Frequency: 184.9972,
		// },
		// {
		// 	Name:      "G3",
		// 	Frequency: 195.9978,
		// },
		// {
		// 	Name:      "G#3",
		// 	Frequency: 207.6523,
		// },
		// {
		// 	Name:      "A3",
		// 	Frequency: 220.0000,
		// },
		// {
		// 	Name:      "A#3",
		// 	Frequency: 233.0819,
		// },
		// {
		// 	Name:      "H3",
		// 	Frequency: 246.9417,
		// },
		{
			Name:      "C4",
			Frequency: 261.6256,
		},
		{
			Name:      "C#4",
			Frequency: 277.1826,
		},
		{
			Name:      "D4",
			Frequency: 293.6648,
		},
		{
			Name:      "D#4",
			Frequency: 311.1270,
		},
		{
			Name:      "E4",
			Frequency: 329.6276,
		},
		{
			Name:      "F4",
			Frequency: 349.2282,
		},
		{
			Name:      "F#4",
			Frequency: 369.9944,
		},
		{
			Name:      "G4",
			Frequency: 391.9954,
		},
		{
			Name:      "G#4",
			Frequency: 415.3047,
		},
		{
			Name:      "A4",
			Frequency: 440.0000,
		},
		{
			Name:      "A#4",
			Frequency: 466.1638,
		},
		{
			Name:      "H4",
			Frequency: 493.8833,
		},
		{
			Name:      "C5",
			Frequency: 523.2511,
		},
		{
			Name:      "C#5",
			Frequency: 554.3653,
		},
		{
			Name:      "D5",
			Frequency: 587.3295,
		},
		{
			Name:      "D#5",
			Frequency: 622.2540,
		},
		{
			Name:      "E5",
			Frequency: 659.2551,
		},
		{
			Name:      "F5",
			Frequency: 698.4565,
		},
		{
			Name:      "F#5",
			Frequency: 739.9888,
		},
		{
			Name:      "G5",
			Frequency: 783.9909,
		},
		{
			Name:      "G#5",
			Frequency: 830.6094,
		},
		{
			Name:      "A5",
			Frequency: 880.0000,
		},
		{
			Name:      "A#5",
			Frequency: 932.3275,
		},
		{
			Name:      "H5",
			Frequency: 987.7666,
		},
		{
			Name:      "C6",
			Frequency: 1046.5023,
		},
		{
			Name:      "C#6",
			Frequency: 1108.7305,
		},
		{
			Name:      "D6",
			Frequency: 1174.6591,
		},
		{
			Name:      "D#6",
			Frequency: 1244.5079,
		},
		{
			Name:      "E6",
			Frequency: 1318.5102,
		},
		// {
		// 	Name:      "F6",
		// 	Frequency: 1396.9129,
		// },
		// {
		// 	Name:      "F#6",
		// 	Frequency: 1479.9777,
		// },
		// {
		// 	Name:      "G6",
		// 	Frequency: 1567.9817,
		// },
		// {
		// 	Name:      "G#6",
		// 	Frequency: 1661.2188,
		// },
		// {
		// 	Name:      "A6",
		// 	Frequency: 1760.0000,
		// },
		// {
		// 	Name:      "A#6",
		// 	Frequency: 1864.6550,
		// },
		// {
		// 	Name:      "H6",
		// 	Frequency: 1975.5332,
		// },
	}

	return notes
}

type NoteStruct struct {
	Name      string
	Frequency float64
}
