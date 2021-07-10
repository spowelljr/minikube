package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"image/color"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

type data struct {
	Date   time.Time `json:"date"`
	Cmd    float64   `json:"cmd"`
	Api    float64   `json:"api"`
	K8s    float64   `json:"k8s"`
	DnsSvc float64   `json:"dnsSvc"`
	App    float64   `json:"app"`
	DnsAns float64   `json:"dnsAns"`
	Total  float64   `json:"total"`
}

type thing struct {
	Data []data `json:"data"`
}

func main() {
	csvPath := flag.String("csv", "", "path to the CSV file")
	chartPath := flag.String("output", "", "path to output the chart to")
	flag.Parse()

	qq := readInCSV(*csvPath)
	d := readData()
	d.Data = append(d.Data, qq)
	updateJSON(d)
	createChart(d, *chartPath)
}

func readInCSV(csvPath string) data {
	f, err := os.Open(csvPath)
	if err != nil {
		log.Fatal(err)
	}

	var cmd, api, k8s, dnsSvc, app, dnsAns float64
	kk := []*float64{&cmd, &api, &k8s, &dnsSvc, &app, &dnsAns}
	count := 0

	r := csv.NewReader(f)
	for {
		d, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		// skip the first line of the CSV file
		if d[0] == "name" {
			continue
		}

		values := []float64{}

		// 8-13 contain the run results
		for i := 8; i <= 13; i++ {
			v, err := strconv.ParseFloat(d[i], 64)
			if err != nil {
				log.Fatal(err)
			}
			values = append(values, v)
		}
		count++
		kk := []*float64{&cmd, &api, &k8s, &dnsSvc, &app, &dnsAns}
		for i, r := range kk {
			*r += values[i]
		}
	}

	var total float64
	for _, r := range kk {
		*r /= float64(count)
		total += *r
	}

	return data{time.Now(), cmd, api, k8s, dnsSvc, app, dnsAns, total}
}

func readData() *thing {
	c, err := os.ReadFile("data.json")
	if err != nil {
		log.Fatal(err)
	}

	t := &thing{}
	if err := json.Unmarshal(c, t); err != nil {
		log.Fatal(err)
	}

	return t
}

func updateJSON(h *thing) {
	d, err := json.Marshal(h)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile("data2.json", d, 0600); err != nil {
		log.Fatal(err)
	}
}

func createChart(t *thing, chartPath string) {
	n := len(t.Data)
	cmdPts := make(plotter.XYs, n)
	apiPts := make(plotter.XYs, n)
	k8sPts := make(plotter.XYs, n)
	dnsSvcPts := make(plotter.XYs, n)
	appPts := make(plotter.XYs, n)
	dnsAnsPts := make(plotter.XYs, n)
	totalPts := make(plotter.XYs, n)

	for i, d := range t.Data {
		pts := []struct {
			pts   *plotter.XYs
			value float64
		}{
			{&cmdPts, d.Cmd},
			{&apiPts, d.Api},
			{&k8sPts, d.K8s},
			{&dnsSvcPts, d.DnsSvc},
			{&appPts, d.App},
			{&dnsAnsPts, d.DnsAns},
			{&totalPts, d.Total},
		}
		date := float64(d.Date.Unix())
		for _, pt := range pts {
			(*pt.pts)[i].Y = pt.value
			(*pt.pts)[i].X = date
		}
	}

	p := plot.New()
	p.Title.Text = "time-to-k8s"
	p.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02"}
	p.Y.Label.Text = "time (seconds)"
	p.Add(plotter.NewGrid())
	p.Legend.Top = true
	p.Y.Max = 95

	cmdLine, cmdPoints := newLine(cmdPts, color.RGBA{R: 255, A: 255})
	apiLine, apiPoints := newLine(apiPts, color.RGBA{G: 255, A: 255})
	k8sLine, k8sPoints := newLine(k8sPts, color.RGBA{B: 255, A: 255})
	dnsSvcLine, dnsSvcPoints := newLine(dnsSvcPts, color.RGBA{R: 255, B: 255, A: 255})
	appLine, appPoints := newLine(appPts, color.RGBA{R: 255, G: 255, A: 255})
	dnsAnsLine, dnsAnsPoints := newLine(dnsAnsPts, color.RGBA{G: 255, B: 255, A: 255})
	totalLine, totalPoints := newLine(totalPts, color.RGBA{B: 255, R: 140, A: 255})

	legend := []struct {
		label string
		line  *plotter.Line
	}{
		{"Command Exec", cmdLine},
		{"API Server Answering", apiLine},
		{"Kubernetes SVC", k8sLine},
		{"DNS SVC", dnsSvcLine},
		{"App Running", appLine},
		{"DNS Answering", dnsAnsLine},
	}

	for _, leg := range legend {
		p.Legend.Add(leg.label, leg.line)
	}

	p.Add(cmdLine, cmdPoints)
	p.Add(apiLine, apiPoints)
	p.Add(k8sLine, k8sPoints)
	p.Add(dnsSvcLine, dnsSvcPoints)
	p.Add(appLine, appPoints)
	p.Add(dnsAnsLine, dnsAnsPoints)
	p.Add(totalLine, totalPoints)

	if err := p.Save(12*vg.Inch, 8*vg.Inch, chartPath); err != nil {
		log.Fatal(err)
	}
}

func newLine(pts plotter.XYs, lineColor color.RGBA) (*plotter.Line, *plotter.Scatter) {
	line, points, err := plotter.NewLinePoints(pts)
	if err != nil {
		log.Fatal(err)
	}
	line.Color = lineColor
	points.Shape = draw.CircleGlyph{}
	points.Color = lineColor

	return line, points
}
