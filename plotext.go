// Package plotext implements extensions and common utilities for gonum/plot.
package plotext

import (
	"encoding/binary"
	"image/color"
	"log"
	"math"
	"os"
	"slices"

	"github.com/dustin/go-humanize"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// QuantizedLine is a plotter.Line derivative that aggregates line points
// quantized into buckets of 1 vg.Point wide when drawing onto a canvas. It
// assumes that the points are more or less evenly distributed along x, which is
// sufficient for most data acquisition sources.
//
// TODO: make it not assume that. Aggregation needs to happen in the x-domain
// (float) rather than the i-domain (sample index).
type QuantizedLine struct {
	*plotter.Line
}

func aggregate(xyer plotter.XYer, n int) (mins, maxes plotter.XYs) {
	mins = make(plotter.XYs, 0, n)
	maxes = make(plotter.XYs, 0, n)

	l := xyer.Len()
	ys := make([]float64, int(math.Ceil(float64(l)/float64(n))))
	di := 0
	x := 0.0

	for i := 0; i < l; {
		x, _ = xyer.XY(i)
		for di = 0; di < len(ys); di++ {
			if i >= l {
				break
			}
			_, ys[di] = xyer.XY(i)
			i++
		}
		mins = append(mins, plotter.XY{X: x, Y: slices.Min(ys)})
		maxes = append(maxes, plotter.XY{X: x, Y: slices.Max(ys)})
	}

	return mins, maxes
}

// Plot draws the data to a `draw.Canvas.`
//
//   - If there are more than 2 data points per Canvas Point of width, the data
//     is first aggregated into buckets per width Point before plotting the
//     bounding min and max lines with an area fill in between using the line
//     color with half the opacity.
//   - Otherwise, the Line is plotted as-is.
func (ql *QuantizedLine) Plot(c draw.Canvas, plt *plot.Plot) {
	dx := int(c.Max.X - c.Min.X)

	if ql.Line.XYs.Len() <= dx*2 {
		ql.Line.Plot(c, plt)
		return
	}

	mins, maxes := aggregate(ql.Line.XYs, dx)

	slices.Reverse(mins)

	verts := append(maxes, mins...)

	poly, err := plotter.NewPolygon(verts)
	if err != nil {
		log.Fatal(err)
	}

	r, g, b, a := ql.Line.Color.RGBA()

	poly.Color = color.NRGBA64{
		R: uint16(r),
		G: uint16(g),
		B: uint16(b),
		A: uint16(a / 2),
	}

	poly.LineStyle.Color = color.Transparent

	poly.Plot(c, plt)

	ql.Line.XYs = maxes
	ql.Line.Plot(c, plt)
	ql.Line.XYs = mins
	ql.Line.Plot(c, plt)
}

// SampleBuffer represents a time-series measurement buffer or trace from a test
// instrument with a fixed sample rate. It implements plotter.XYer using the
// sample rate to calculate X-values in seconds starting from 0.
type SampleBuffer struct {
	Samples    []float64
	SampleRate float64 // samples per second
}

// Len returns the number of x, y pairs.
func (s *SampleBuffer) Len() int {
	return len(s.Samples)
}

// XY returns an x, y pair.
func (s *SampleBuffer) XY(i int) (x float64, y float64) {
	return float64(i) / s.SampleRate, s.Samples[i]
}

// LoadSampleBuffer loads a big-endian binary file containing `size` float64
// values from disk and constructs a SampleBuffer object with the given sample
// rate `fs`.
func LoadSampleBuffer(path string, size int, fs float64) *SampleBuffer {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	p := make([]float64, size)

	err = binary.Read(f, binary.BigEndian, &p)
	if err != nil {
		log.Fatal(err)
	}

	return &SampleBuffer{
		Samples:    p,
		SampleRate: fs,
	}
}

type AutoTicker struct {
	Dim vg.Length
}

// Ticks returns Ticks in a specified range
func (t AutoTicker) Ticks(min float64, max float64) []plot.Tick {

	dim := t.Dim
	if dim == 0 {
		dim = 800
	}

	// select an appropriate power of 10 minor tick interval
	const (
		targetTickPitch  = font.Inch / 5
		targetLabelPitch = font.Inch
	)

	targetTickCount := float64(dim / targetTickPitch)       // ul
	targetMinorTickSpacing := (max - min) / targetTickCount // data units
	// rounded to nearest power of 10
	selectedMag := math.Round(math.Log10(float64(targetMinorTickSpacing))) // log10 data units
	selectedMinorTickSpacing := math.Pow10(int(selectedMag))               // data units
	selectedMinorTickCount := float64(max-min) / selectedMinorTickSpacing  // ul
	// selectedMinorTickPitch := dim / vg.Length(selectedMinorTickCount)      // canvas units

	// major ticks at 2, 5, or 10 minor tick intervals to achieve as close to 1 label per inch as possible
	targetMajorTickCount := float64(dim / targetLabelPitch) // index units
	targetMajorTickInterval := math.Round(selectedMinorTickCount / targetMajorTickCount)
	selectedMajorTickInterval := 2
	if targetMajorTickInterval > 5 {
		selectedMajorTickInterval = 10
	} else if targetMajorTickInterval > 2 {
		selectedMajorTickInterval = 5
	}

	minTickIndex := int(math.Floor(min / selectedMinorTickSpacing))
	maxTickIndex := int(math.Ceil(max / selectedMinorTickSpacing))

	/*
		vals := []struct {
			name string
			val  interface{}
		}{
			{"targetTickCount", targetTickCount},
			{"targetMinorTickSpacing", targetMinorTickSpacing},
			{"selectedMag", selectedMag},
			{"selectedMinorTickSpacing", selectedMinorTickSpacing},
			{"selectedTickCount", selectedMinorTickCount},
			{"selectedTickPitch", selectedMinorTickPitch},
			{"targetMajorTickInterval", targetMajorTickInterval},
			{"selectedMajorTickInterval", selectedMajorTickInterval},
			{"minTickIndex", minTickIndex},
			{"maxTickIndex", maxTickIndex},
		}

		for _, v := range vals {
			fmt.Println(v.name, v.val)
		}
	*/

	ret := make([]plot.Tick, 0, maxTickIndex-minTickIndex+1)
	for i := minTickIndex; i <= maxTickIndex; i++ {
		t := plot.Tick{
			Value: float64(i) * selectedMinorTickSpacing,
		}

		if i%selectedMajorTickInterval == 0 {
			// todo:
			// * trim to significant figures
			// * if largest value is [1, 1000): no suffix
			// * others: add SI prefix with 3 sigfigs max
			t.Label = humanize.SI(t.Value, "")
		}
		ret = append(ret, t)
	}

	return ret

	// return nil
}
