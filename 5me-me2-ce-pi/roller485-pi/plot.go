package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"strings"
)

const (
	imgW   = 800
	imgH   = 400
	margin = 10
)

func savePlot(csvName string, data []row) {
	if len(data) < 2 {
		return
	}

	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	fill(img, color.RGBA{255, 255, 255, 255})

	halfH := imgH / 2
	blue   := color.RGBA{31, 119, 180, 255}
	red    := color.RGBA{214, 39, 40, 255}
	gray   := color.RGBA{160, 160, 160, 255}
	orange := color.RGBA{255, 152, 80, 255}

	minT, maxT := data[0].t, data[len(data)-1].t
	// 上半分: omega_ref と speed を同一スケールで描画
	minS1, maxS1 := minmax(data, func(r row) float64 { return r.omegaRef })
	minS2, maxS2 := minmax(data, func(r row) float64 { return r.speed })
	minS, maxS := minS1, maxS1
	if minS2 < minS { minS = minS2 }
	if maxS2 > maxS { maxS = maxS2 }
	// 下半分: current_cmd と current_meas を同一スケールで描画
	minC1, maxC1 := minmax(data, func(r row) float64 { return r.currentCmd })
	minC2, maxC2 := minmax(data, func(r row) float64 { return r.currentMeas })
	minC, maxC := minC1, maxC1
	if minC2 < minC { minC = minC2 }
	if maxC2 > maxC { maxC = maxC2 }

	scaleX := func(t float64) int {
		return margin + int((t-minT)/(maxT-minT)*float64(imgW-2*margin))
	}
	scaleY := func(v, mn, mx float64, top, h int) int {
		if mx == mn {
			return top + h/2
		}
		return top + margin + int((1-(v-mn)/(mx-mn))*float64(h-2*margin))
	}

	for i := 1; i < len(data); i++ {
		p, q := data[i-1], data[i]
		// 上段: ref → speed の順 (主役の speed を上に乗せる)
		drawLine(img, scaleX(p.t), scaleY(p.omegaRef, minS, maxS, 0, halfH),
			scaleX(q.t), scaleY(q.omegaRef, minS, maxS, 0, halfH), gray)
		drawLine(img, scaleX(p.t), scaleY(p.speed, minS, maxS, 0, halfH),
			scaleX(q.t), scaleY(q.speed, minS, maxS, 0, halfH), blue)
		// 下段: cmd → meas の順
		drawLine(img, scaleX(p.t), scaleY(p.currentCmd, minC, maxC, halfH, halfH),
			scaleX(q.t), scaleY(q.currentCmd, minC, maxC, halfH, halfH), orange)
		drawLine(img, scaleX(p.t), scaleY(p.currentMeas, minC, maxC, halfH, halfH),
			scaleX(q.t), scaleY(q.currentMeas, minC, maxC, halfH, halfH), red)
	}

	pngName := strings.TrimSuffix(csvName, ".csv") + ".png"
	out, err := os.Create(pngName)
	if err != nil {
		log.Printf("プロット保存エラー: %v", err)
		return
	}
	defer out.Close()
	png.Encode(out, img)
	fmt.Printf("プロット: %s\n", pngName)
}

func fill(img *image.RGBA, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx, dy := abs(x1-x0), abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 { sx = -1 }
	if y0 > y1 { sy = -1 }
	e := dx - dy
	for {
		img.SetRGBA(x0, y0, c)
		if x0 == x1 && y0 == y1 { break }
		e2 := 2 * e
		if e2 > -dy { e -= dy; x0 += sx }
		if e2 < dx  { e += dx; y0 += sy }
	}
}

func abs(x int) int {
	if x < 0 { return -x }
	return x
}

func minmax(data []row, f func(row) float64) (float64, float64) {
	mn, mx := f(data[0]), f(data[0])
	for _, r := range data[1:] {
		if v := f(r); v < mn { mn = v }
		if v := f(r); v > mx { mx = v }
	}
	return mn, mx
}
