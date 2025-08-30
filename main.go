package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
	"io/ioutil" // Consider using os.ReadFile for Go 1.16+
	"log"
	"os"
)

var (
	codeText string
	index    int
	speed    = 3 // 每次敲键盘输出几个字符
)

// 渲染一个字符（自动处理宽度）
// This function is defined but not used in the current code.
func drawCharacter(s tcell.Screen, x, y int, r rune, style tcell.Style) int {
	w := runewidth(r)
	s.SetContent(x, y, r, nil, style)
	return w
}

// 获取字符宽度（ASCII: 1，中文：2）
func runewidth(r rune) int {
	g := uniseg.NewGraphemes(string(r))
	for g.Next() {
		return g.Width()
	}
	return 1
}

// 居中显示一条消息
func showCentered(s tcell.Screen, msg string, style tcell.Style) {
	w, h := s.Size()
	x := (w - len(msg)) / 2
	y := h / 2
	for i, r := range msg {
		s.SetContent(x+i, y, r, nil, style)
	}
	s.Show()
}

func main() {
	// 加载 kernel.txt
	data, err := ioutil.ReadFile("kernel.txt") // os.ReadFile is preferred for Go 1.16+
	if err != nil {
		fmt.Println("无法读取 kernel.txt:", err)
		os.Exit(1)
	}
	codeText = string(data)

	// 初始化屏幕
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("创建屏幕失败: %v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("初始化屏幕失败: %v", err)
	}
	defer s.Fini()

	// 初始清屏
	s.Clear()
	style := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorBlack)

mainloop:
	for {
		// 等待键盘事件
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEsc, tcell.KeyCtrlC:
				break mainloop

			case tcell.KeyCtrlA: // Access Granted
				s.Clear()
				showCentered(s, "ACCESS GRANTED", style.Bold(true))
				continue
			case tcell.KeyCtrlD: // Access Denied
				s.Clear()
				deniedStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack).Bold(true)
				showCentered(s, "ACCESS DENIED", deniedStyle)
				continue

			default:
				// 每按任意键，输出代码 speed 个字符
				w, h := s.Size() // Get current screen dimensions
				y, x := 0, 0     // 光标位置

				// Redraw already typed characters to maintain state
				for i := 0; i < index; i++ {
					ch := rune(codeText[i])
					if ch == '\n' {
						y++
						x = 0
					} else {
						charWidth := runewidth(ch) // Use character width
						// Check for screen wrap before setting content for multi-width chars
						if x+charWidth > w {
							x = 0
							y++
						}
						if y >= h { // If screen full, clear and reset position
							s.Clear()
							y, x = 0, 0
						}
						s.SetContent(x, y, ch, nil, style)
						x += charWidth
					}
				}

				// Output new segment
				next := index + speed
				if next > len(codeText) {
					next = len(codeText)
				}
				for i := index; i < next; i++ {
					ch := rune(codeText[i])
					if ch == '\n' {
						y++
						x = 0
					} else {
						charWidth := runewidth(ch) // 计算字符实际宽度
						// Check for screen wrap before setting content for multi-width chars
						if x+charWidth > w { // Changed wScreen to w
							x = 0
							y++
						}
						if y >= h { // Changed hScreen to h
							s.Clear()
							y, x = 0, 0
						}
						s.SetContent(x, y, ch, nil, style)
						x += charWidth
					}
				}

				index = next
				// 如果读到结尾，循环回去
				if index >= len(codeText) {
					index = 0
					s.Clear()
				}
				s.Show()
			}
		}
	}
}
