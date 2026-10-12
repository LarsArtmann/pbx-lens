package main

import (
	"context"
	"os"

	"github.com/larsartmann/templ-components/display"
)

func main() {
	bars := []display.BarChartBar{
		{Label: "mon", Value: 3},
		{Label: "tue", Value: 7},
		{Label: "wed", Value: 1},
	}

	// Variant A: exactly what bar_chart.go:93-96 tells consumers to do
	// ("Height ... Essential for vertical charts").
	a := display.BarChartProps{Orient: display.BarVertical, Bars: bars, Height: "8rem"}

	// Variant B: the same plus the undocumented items-stretch override.
	b := a
	b.Class = "!items-stretch"

	f, err := os.Create("/tmp/barchart-repro.html")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.WriteString(`<!doctype html><html><head><script src="https://cdn.tailwindcss.com"></script></head><body style="padding:2rem">
<h3>A: Height only (doc-sufficient)</h3><div id="a">`)
	if err := display.BarChart(a).Render(context.Background(), f); err != nil {
		panic(err)
	}
	f.WriteString(`</div><h3>B: Height + !items-stretch (workaround)</h3><div id="b">`)
	if err := display.BarChart(b).Render(context.Background(), f); err != nil {
		panic(err)
	}
	f.WriteString(`</div></body></html>`)
}
