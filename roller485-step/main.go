package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.bug.st/serial"
)

type row struct{ t, speed, current float64 }

var saveDir string

var stdin = bufio.NewReader(os.Stdin)

func prompt(label string) string {
	fmt.Print(label + ": ")
	line, _ := stdin.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptInt(label string) int {
	for {
		s := prompt(label)
		n, err := strconv.Atoi(s)
		if err == nil {
			return n
		}
		fmt.Println("整数で入力してください")
	}
}

func main() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("実行パス取得エラー: %v", err)
	}
	saveDir = filepath.Dir(exe)

	// ポート選択
	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatalf("ポート一覧取得エラー: %v", err)
	}
	if len(ports) == 0 {
		log.Fatal("利用可能なポートがありません")
	}
	fmt.Println("利用可能なポート:")
	for i, p := range ports {
		fmt.Printf("  [%d] %s\n", i, p)
	}
	var idx int
	for {
		idx = promptInt("番号")
		if idx >= 0 && idx < len(ports) {
			break
		}
		fmt.Printf("0〜%d の番号を入力してください\n", len(ports)-1)
	}
	portName := ports[idx]

	port, err := serial.Open(portName, &serial.Mode{BaudRate: 115200})
	if err != nil {
		log.Fatalf("ポートを開けません: %v", err)
	}
	defer port.Close()
	port.SetReadTimeout(100 * time.Millisecond)

	// Ctrl-C でモーター停止してから終了
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n停止中...")
		port.Write([]byte("stop\n"))
		os.Exit(0)
	}()

	lines := make(chan string, 256)
	go func() {
		var buf []byte
		tmp := make([]byte, 256)
		for {
			n, err := port.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
				for {
					idx := bytes.IndexByte(buf, '\n')
					if idx < 0 {
						break
					}
					line := strings.TrimSpace(string(buf[:idx]))
					buf = buf[idx+1:]
					if line != "" {
						lines <- line
					}
				}
			}
			if err != nil {
				return
			}
			// n==0, err==nil はタイムアウト → ループ継続
		}
	}()

	// 起動メッセージを表示
	startup := time.After(2 * time.Second)
loop:
	for {
		select {
		case line := <-lines:
			fmt.Println(line)
		case <-startup:
			break loop
		}
	}

	// 計測ループ
	for {
		fmt.Println()
		iPre  := promptInt("I_pre  [mA]")
		iStep := promptInt("I_step [mA]")
		tHold := promptInt("T_hold [ms]")

		filename := filepath.Join(saveDir, time.Now().Format("20060102150405")+".csv")
		f, err := os.Create(filename)
		if err != nil {
			log.Printf("ファイル作成エラー: %v", err)
			continue
		}

		drainLines(lines)
		if err := port.ResetInputBuffer(); err != nil {
			log.Printf("バッファクリアエラー: %v", err)
		}
		fmt.Fprintf(port, "%d %d %d\n", iPre, iStep, tHold)
		data := collectData(lines, f, time.Duration(2*tHold+500)*time.Millisecond)
		f.Close()
		fmt.Printf("保存: %s\n", filename)
		savePlot(filename, data)
	}
}

func drainLines(lines <-chan string) {
	for {
		select {
		case <-lines:
		default:
			return
		}
	}
}

func collectData(lines <-chan string, f *os.File, duration time.Duration) []row {
	var data []row
	deadline := time.After(duration)
	for {
		select {
		case line := <-lines:
			fmt.Fprintln(f, line)
			parts := strings.Split(line, ",")
			if len(parts) == 3 {
				t, e1 := strconv.ParseFloat(parts[0], 64)
				s, e2 := strconv.ParseFloat(parts[1], 64)
				c, e3 := strconv.ParseFloat(parts[2], 64)
				if e1 == nil && e2 == nil && e3 == nil {
					data = append(data, row{t, s, c})
				}
			}
		case <-deadline:
			return data
		}
	}
}
