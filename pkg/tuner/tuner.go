package tuner

import (
	"fmt"
	"math"
	"math/cmplx"
	"sync"

	"github.com/andrepxx/go-dsp-guitar/circular"
	"github.com/andrepxx/go-dsp-guitar/fft"
)

/*
 * Global constants.
 */
const (
	NUM_SAMPLES = 96000
)

/*
 * Data structure representing a musical note.
 */
type NoteStruct struct {
	Name      string
	Frequency float64
}

/*
 * Data structure representing the result of a spectral analysis.
 */
type Result struct {
	cents          int8
	frequency      float64
	note           string
	SubCorrelation []float64
	NoteValues     []NoteValue
}

/*
 * The result of a spectral analysis.
 */

/*
 * Data structure representing a tuner.
 */
type Tuner struct {
	notes            []NoteStruct
	mutexBuffer      sync.RWMutex
	buffer           circular.Buffer
	sampleRate       uint32
	mutexAnalyze     sync.Mutex
	fourierTransform fft.FourierTransform
	bufCorrelation   []float64
	bufFFT           []complex128
}

/*
 * A chromatic instrument tuner.
 */

/*
 * Generates a list of notes and their frequencies.
 *
 * f(n) = 2^(n / 12) * 440
 *
 * Where n is the number of half-tone steps relative to A4.
 */
func generateNotes() []NoteStruct {

	/*
	 * Create a list of appropriate notes.
	 */
	notes := []NoteStruct{
		NoteStruct{
			Name:      "H1",
			Frequency: 61.7354,
		},
		NoteStruct{
			Name:      "C2",
			Frequency: 65.4064,
		},
		NoteStruct{
			Name:      "C#2",
			Frequency: 69.2957,
		},
		NoteStruct{
			Name:      "D2",
			Frequency: 73.4162,
		},
		NoteStruct{
			Name:      "D#2",
			Frequency: 77.7817,
		},
		NoteStruct{
			Name:      "E2",
			Frequency: 82.4069,
		},
		NoteStruct{
			Name:      "F2",
			Frequency: 87.3071,
		},
		NoteStruct{
			Name:      "F#2",
			Frequency: 92.4986,
		},
		NoteStruct{
			Name:      "G2",
			Frequency: 97.9989,
		},
		NoteStruct{
			Name:      "G#2",
			Frequency: 103.8262,
		},
		NoteStruct{
			Name:      "A2",
			Frequency: 110.0000,
		},
		NoteStruct{
			Name:      "A#2",
			Frequency: 116.5409,
		},
		NoteStruct{
			Name:      "H2",
			Frequency: 123.4708,
		},
		NoteStruct{
			Name:      "C3",
			Frequency: 130.8128,
		},
		NoteStruct{
			Name:      "C#3",
			Frequency: 138.5913,
		},
		NoteStruct{
			Name:      "D3",
			Frequency: 146.8324,
		},
		NoteStruct{
			Name:      "D#3",
			Frequency: 155.5635,
		},
		NoteStruct{
			Name:      "E3",
			Frequency: 164.8138,
		},
		NoteStruct{
			Name:      "F3",
			Frequency: 174.6141,
		},
		NoteStruct{
			Name:      "F#3",
			Frequency: 184.9972,
		},
		NoteStruct{
			Name:      "G3",
			Frequency: 195.9978,
		},
		NoteStruct{
			Name:      "G#3",
			Frequency: 207.6523,
		},
		NoteStruct{
			Name:      "A3",
			Frequency: 220.0000,
		},
		NoteStruct{
			Name:      "A#3",
			Frequency: 233.0819,
		},
		NoteStruct{
			Name:      "H3",
			Frequency: 246.9417,
		},
		NoteStruct{
			Name:      "C4",
			Frequency: 261.6256,
		},
		NoteStruct{
			Name:      "C#4",
			Frequency: 277.1826,
		},
		NoteStruct{
			Name:      "D4",
			Frequency: 293.6648,
		},
		NoteStruct{
			Name:      "D#4",
			Frequency: 311.1270,
		},
		NoteStruct{
			Name:      "E4",
			Frequency: 329.6276,
		},
		NoteStruct{
			Name:      "F4",
			Frequency: 349.2282,
		},
		NoteStruct{
			Name:      "F#4",
			Frequency: 369.9944,
		},
		NoteStruct{
			Name:      "G4",
			Frequency: 391.9954,
		},
		NoteStruct{
			Name:      "G#4",
			Frequency: 415.3047,
		},
		NoteStruct{
			Name:      "A4",
			Frequency: 440.0000,
		},
		NoteStruct{
			Name:      "A#4",
			Frequency: 466.1638,
		},
		NoteStruct{
			Name:      "H4",
			Frequency: 493.8833,
		},
		NoteStruct{
			Name:      "C5",
			Frequency: 523.2511,
		},
		NoteStruct{
			Name:      "C#5",
			Frequency: 554.3653,
		},
		NoteStruct{
			Name:      "D5",
			Frequency: 587.3295,
		},
		NoteStruct{
			Name:      "D#5",
			Frequency: 622.2540,
		},
		NoteStruct{
			Name:      "E5",
			Frequency: 659.2551,
		},
		NoteStruct{
			Name:      "F5",
			Frequency: 698.4565,
		},
		NoteStruct{
			Name:      "F#5",
			Frequency: 739.9888,
		},
		NoteStruct{
			Name:      "G5",
			Frequency: 783.9909,
		},
		NoteStruct{
			Name:      "G#5",
			Frequency: 830.6094,
		},
		NoteStruct{
			Name:      "A5",
			Frequency: 880.0000,
		},
		NoteStruct{
			Name:      "A#5",
			Frequency: 932.3275,
		},
		NoteStruct{
			Name:      "H5",
			Frequency: 987.7666,
		},
		NoteStruct{
			Name:      "C6",
			Frequency: 1046.5023,
		},
		NoteStruct{
			Name:      "C#6",
			Frequency: 1108.7305,
		},
		NoteStruct{
			Name:      "D6",
			Frequency: 1174.6591,
		},
		NoteStruct{
			Name:      "D#6",
			Frequency: 1244.5079,
		},
		NoteStruct{
			Name:      "E6",
			Frequency: 1318.5102,
		},
		NoteStruct{
			Name:      "F6",
			Frequency: 1396.9129,
		},
		NoteStruct{
			Name:      "F#6",
			Frequency: 1479.9777,
		},
		NoteStruct{
			Name:      "G6",
			Frequency: 1567.9817,
		},
		NoteStruct{
			Name:      "G#6",
			Frequency: 1661.2188,
		},
		NoteStruct{
			Name:      "A6",
			Frequency: 1760.0000,
		},
		NoteStruct{
			Name:      "A#6",
			Frequency: 1864.6550,
		},
		NoteStruct{
			Name:      "H6",
			Frequency: 1975.5332,
		},
	}

	return notes
}

/*
 * Find the maximum value in a buffer.
 */
func findMaximum(buf []float64) (float64, int) {
	maxVal := math.Inf(-1)
	maxIdx := int(-1)

	/*
	 * Iterate over the buffer and find the maximum value.
	 */
	for idx, value := range buf {

		/*
		 * If we found a value which is greater than any value we
		 * encountered so far, make it the new candidate.
		 */
		if value > maxVal {
			maxVal = value
			maxIdx = idx
		}

	}

	return maxVal, maxIdx
}

/*
 * Returns the deviation from the reference note in cents.
 */
func (this *Result) Cents() int8 {
	return this.cents
}

/*
 * Returns the fundamental frequency of the signal.
 */
func (this *Result) Frequency() float64 {
	return this.frequency
}

/*
 * Returns the name of the closest note on the chromatic scale.
 */
func (this *Result) Note() string {
	return this.note
}

/*
 * Analyze buffered stream for spectral content.
 */
func (this *Tuner) Analyze() (*Result, error) {
	this.mutexAnalyze.Lock()
	circularBuffer := this.buffer
	bufCorrelation := this.bufCorrelation
	bufCorrrlationLength := len(bufCorrelation)
	bufCorrelationLength64 := uint64(bufCorrrlationLength)
	bufFFT := this.bufFFT
	bufFFTLength := len(bufFFT)
	bufFFTLength64 := uint64(bufFFTLength)
	n := circularBuffer.Length()
	twoN := uint64(2 * n)
	fftSize, _ := fft.NextPowerOfTwo(twoN)

	/*
	 * Ensure that correlation buffer is of correct length.
	 */
	if bufCorrelationLength64 != fftSize {
		bufCorrelation = make([]float64, fftSize)
		this.bufCorrelation = bufCorrelation
	}

	/*
	 * Ensure that FFT buffer is of correct length.
	 */
	if bufFFTLength64 != fftSize {
		bufFFT = make([]complex128, fftSize)
		this.bufFFT = bufFFT
	}

	signalBuffer := bufCorrelation[0:n]
	this.mutexBuffer.RLock()
	sampleRate := this.sampleRate
	err := circularBuffer.Retrieve(signalBuffer)
	this.mutexBuffer.RUnlock()

	/*
	 * Verify that buffer contents could be retrieved.
	 */
	if err != nil {
		msg := err.Error()
		this.mutexAnalyze.Unlock()
		return nil, fmt.Errorf("Failed to retrieve contents of circular buffer: %s", msg)
	} else {
		ft := this.fourierTransform
		tailBuffer := bufCorrelation[n:fftSize]
		fft.ZeroFloat(tailBuffer)
		err = ft.RealFourier(bufCorrelation, bufFFT, fft.SCALING_DEFAULT)

		/*
		 * Verify that the forward FFT was calculated successfully.
		 */
		if err != nil {
			msg := err.Error()
			this.mutexAnalyze.Unlock()
			return nil, fmt.Errorf("Failed to calculate forward FFT: %s", msg)
		} else {

			/*
			 * Multiply each element of the spectrum with its complex conjugate.
			 */
			for i, elem := range bufFFT {
				elemConj := cmplx.Conj(elem)
				bufFFT[i] = elem * elemConj
			}

			err = ft.RealInverseFourier(bufFFT, bufCorrelation, fft.SCALING_DEFAULT)

			/*
			 * Verify that the inverse FFT was calculated successfully.
			 */
			if err != nil {
				msg := err.Error()
				this.mutexAnalyze.Unlock()
				return nil, fmt.Errorf("Failed to calculate inverse FFT: %s", msg)
			} else {
				notes := this.notes
				noteCount := len(notes)
				lastNote := noteCount - 1
				lowFreq := notes[0].Frequency
				highFreq := notes[lastNote].Frequency
				sampleRateFloat := float64(sampleRate)
				lowIdx := int((sampleRateFloat / highFreq) + 0.5)
				lowIdx64 := uint64(lowIdx)

				/*
				 * This might happen when the float value is infinite.
				 */
				if (lowIdx < 0) || (lowIdx64 >= twoN) {
					lowIdx = 0
					lowIdx64 = 0
				}

				highIdx := int((sampleRateFloat / lowFreq) + 0.5)
				highIdx64 := uint64(highIdx)

				/*
				 * This might happen when the float value is infinite.
				 */
				if (highIdx < 0) || (highIdx64 >= twoN) {
					maxIdx := twoN - 1
					highIdx = int(maxIdx)
					highIdx64 = maxIdx
				}

				subCorrelation := bufCorrelation[lowIdx:highIdx]
				maxVal, maxIdx := findMaximum(subCorrelation)
				idx := lowIdx + maxIdx
				idxUp := idx + 1

				/*
				 * Prevent overrun.
				 */
				if idxUp > n {
					idxUp = n
				}

				idxDown := idx - 1

				/*
				 * Prevent underrun.
				 */
				if idxDown < 0 {
					idxDown = 0
				}

				valueLeft := bufCorrelation[idxDown]
				valueRight := bufCorrelation[idxUp]
				idxFloat := float64(idx)
				valueDiff := valueRight - valueLeft
				valueSum := valueRight + valueLeft
				halfDiff := 0.5 * valueDiff
				doubleMaxVal := 2.0 * maxVal
				denominatorDiff := doubleMaxVal - valueSum
				shiftEstimation := halfDiff / denominatorDiff

				/*
				 * Limit shift estimation to plus/minus half a sample.
				 */
				if shiftEstimation < -0.5 {
					shiftEstimation = -0.5
				} else if shiftEstimation > 0.5 {
					shiftEstimation = 0.5
				}

				idxFloat += shiftEstimation
				actualFrequency := sampleRateFloat / idxFloat
				actualNote := "Unknown"
				actualCents := math.Inf(1)
				actualCentsAbs := math.Abs(actualCents)

				/*
				 * Iterate over all notes and find the closest match.
				 */
				noteValues := []NoteValue{}
				for _, note := range notes {
					freq := note.Frequency
					freqRatio := actualFrequency / freq
					diffCents := 1200.0 * math.Log2(freqRatio)
					diffCentsAbs := math.Abs(diffCents)

					noteValues = append(noteValues, NoteValue{
						Note:  note,
						Value: diffCentsAbs,
					})
					/*
					 * If this is the closest we've seen so far, make this the best match.
					 */
					if diffCentsAbs < actualCentsAbs {
						actualNote = note.Name
						actualCents = diffCents
						actualCentsAbs = diffCentsAbs
					}

				}

				actualCentsInfinite := math.IsInf(actualCents, 0)
				actualCentsNaN := math.IsNaN(actualCents)
				actualCentsInt := int8(0)

				/*
				 * If cents are finite, use them.
				 */
				if !(actualCentsInfinite || actualCentsNaN) {
					actualCentsInt = int8(actualCents)
				}

				/*
				 * Create result of signal analysis.
				 */

				copySubCorrelation := make([]float64, len(subCorrelation))
				copy(copySubCorrelation, subCorrelation)
				result := Result{
					cents:          actualCentsInt,
					frequency:      actualFrequency,
					note:           actualNote,
					SubCorrelation: copySubCorrelation,
					NoteValues:     noteValues,
				}

				this.mutexAnalyze.Unlock()
				return &result, nil
			}

		}

	}

}

/*
 * Stream samples for later analysis.
 */
func (this *Tuner) Process(samples []float64, sampleRate uint32) {
	this.mutexBuffer.Lock()
	this.buffer.Enqueue(samples...)
	this.sampleRate = sampleRate
	this.mutexBuffer.Unlock()
}

/*
 * Creates an instrument tuner.
 */
func Create() *Tuner {
	notes := generateNotes()
	buffer := circular.CreateBuffer(NUM_SAMPLES)
	ft := fft.CreateFourierTransform()

	/*
	 * Create data structure for a guitar tuner.
	 */
	t := Tuner{
		notes:            notes,
		buffer:           buffer,
		fourierTransform: ft,
	}

	return &t
}

type NoteValue struct {
	Note  NoteStruct
	Value float64
}
