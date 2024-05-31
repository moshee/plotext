package plotext

import (
	"math"
	"slices"
	"testing"

	"github.com/dustin/go-humanize"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
)

func TestTicker(t *testing.T) {
	table := []struct {
		dim              vg.Length
		min, max         float64
		tickMin, tickMax float64
		tickSpacing      float64
		majorInterval    int
	}{
		{0, 0, 0, 0, 0, 0, 0},
		{0, 0, 1, 0, 1, 0.01, 10},
		{0, -1, 1, -1, 1, 0.1, 2},
		{100, 0, 0.9, 0, 0.9, 0.1, 10},
		{123, 0, 1.5, 0, 1.5, 0.1, 10},
		{80, -1, 1, -1, 1, 1, 10},
		{105, 0.5, 10, 0, 10, 1, 10},
		{305, -1, 0, -1, 0, 0.1, 2},
		{928, -10, 0, -10, 0, 0.1, 10},
		{1294, -12.6, -5, -12.6, -5, 0.1, 5},
	}

	for _, row := range table {
		dut := AutoTicker{row.dim}
		ticks := dut.Ticks(row.min, row.max)
		ex := expectedTicks(row.tickMin, row.tickMax, row.tickSpacing, row.majorInterval)
		if !slices.Equal(ticks, ex) {
			t.Error("---")
			t.Errorf("input: dim=%f min=%f max=%f", row.dim, row.min, row.max)
			t.Errorf("got: %v", ticks)
			t.Errorf("expected: %v", ex)
		}
	}
}

func expectedTicks(min, max, spacing float64, interval int) []plot.Tick {
	if spacing == 0 {
		return []plot.Tick{{Value: min, Label: humanize.SI(min, "")}}
	}
	ret := make([]plot.Tick, 0, int((max-min)/spacing))

	for i := int(math.Round(min / spacing)); i <= int(math.Round(max/spacing)); i++ {
		t := plot.Tick{Value: float64(i) * spacing}
		if i%interval == 0 {
			t.Label = humanize.SI(t.Value, "")
		}
		ret = append(ret, t)
	}
	return ret
}
