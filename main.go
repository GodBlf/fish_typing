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
	index    int // Current position in codeText (where the next characters will be read from)

	speed = 3 // Number of characters to output per key press

	screenLines [][]rune // Buffer to hold characters for each visible line on the screen
	currentLine int      // Current line index within screenLines where new characters are added
	currentCol  int      // Current column index within currentLine where new characters are added

	screenWidth  int // Current width of the tcell screen
	screenHeight int // Current height of the tcell screen
)

// runewidth 获取字符宽度（ASCII: 1，中文：2）
func runewidth(r rune) int {
	g := uniseg.NewGraphemes(string(r))
	for g.Next() {
		return g.Width()
	}
	return 1
}

// showCentered 居中显示一条消息。此函数会清除屏幕并显示消息，不参与滚动。
func showCentered(s tcell.Screen, msg string, style tcell.Style) {
	w, h := s.Size()
	x := (w - len(msg)) / 2
	y := h / 2
	s.Clear() // Clear the screen for the centered message overlay
	for i, r := range msg {
		s.SetContent(x+i, y, r, nil, style)
	}
	s.Show()
}

// redrawScreen 将 screenLines 缓冲区的内容绘制到 tcell 屏幕上。
func redrawScreen(s tcell.Screen, style tcell.Style) {
	s.Clear() // 在重新绘制所有内容之前清除屏幕

	for y, line := range screenLines {
		xOffset := 0
		for _, r := range line {
			charWidth := runewidth(r)
			s.SetContent(xOffset, y, r, nil, style)
			xOffset += charWidth
		}
	}
	s.Show()
}

// initScreenLines 根据 screenHeight 初始化 screenLines 缓冲区，使其包含空行。
func initScreenLines(height int) {
	screenLines = make([][]rune, height)
	for i := range screenLines {
		screenLines[i] = []rune{} // 将每行初始化为空的 rune 切片
	}
	// currentLine 和 currentCol 由调用者管理（例如，addCharactersToBuffer 或 resize 处理程序）
}

// addCharactersToBuffer 模拟从 codeText 的 startTextIndex 位置开始“键入” `count` 个字符，
// 更新 screenLines, currentLine, currentCol，并处理行包装和滚动。
// 它返回处理后在 codeText 中的新文本索引。
func addCharactersToBuffer(startTextIndex int, count int) int {
	endTextIndex := startTextIndex + count
	if endTextIndex > len(codeText) {
		endTextIndex = len(codeText)
	}

	for i := startTextIndex; i < endTextIndex; i++ {
		ch := rune(codeText[i])

		if ch == '\n' {
			currentLine++
			currentCol = 0
		} else {
			charWidth := runewidth(ch)
			// 检查是否需要行包装
			if currentCol+charWidth > screenWidth {
				currentLine++
				currentCol = 0
			}

			// 如果 currentLine 超出 screenHeight，则执行滚动
			if currentLine >= screenHeight {
				// 移除第一行（向上滚动）
				// 注意：如果 screenLines 长度为0或1，且 screenHeight为1，这里需要小心
				if len(screenLines) > 0 {
					screenLines = screenLines[1:]
				}
				// 在底部添加一个新空行
				screenLines = append(screenLines, []rune{})
				currentLine = screenHeight - 1 // 保持 currentLine 指向最后（新添加的）一行
			}

			// 确保目标行切片存在且可寻址
			// 这个检查处理了 screenHeight 可能为0或很小的情况，
			// 或者如果某个逻辑错误导致 currentLine 超出缓冲区范围。
			for len(screenLines) <= currentLine {
				screenLines = append(screenLines, []rune{})
			}
			screenLines[currentLine] = append(screenLines[currentLine], ch)
			currentCol += charWidth
		}
	}
	return endTextIndex // 返回在 codeText 中处理完成的索引
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

	// 初始样式、屏幕尺寸和缓冲区设置
	style := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorBlack)
	screenWidth, screenHeight = s.Size()
	initScreenLines(screenHeight)
	redrawScreen(s, style) // 绘制初始空屏幕

mainloop:
	for {
		// 等待键盘事件
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			newWidth, newHeight := s.Size()

			// 保存当前的全局文本索引
			oldGlobalTextIndex := index

			// 重置屏幕尺寸
			screenWidth = newWidth
			screenHeight = newHeight

			// 重置缓冲区内光标位置以进行重排
			currentLine = 0
			currentCol = 0
			initScreenLines(screenHeight) // 清空并为新高度重新初始化缓冲区

			// 重排所有先前键入的字符，直到 oldGlobalTextIndex
			// addCharactersToBuffer 被重复调用以模拟从 codeText 开头到 oldGlobalTextIndex 的键入。
			// 我们使用 reflowTextIndex 来跟踪重排过程中 codeText 的进度。
			reflowTextIndex := 0
			for reflowTextIndex < oldGlobalTextIndex {
				// 为了准确重排，每次添加一个字符。
				reflowTextIndex = addCharactersToBuffer(reflowTextIndex, 1)
			}
			index = oldGlobalTextIndex // 重排后恢复全局文本索引

			s.Sync()
			redrawScreen(s, style)

		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEsc, tcell.KeyCtrlC:
				break mainloop

			case tcell.KeyCtrlA: // Access Granted
				showCentered(s, "ACCESS GRANTED", style.Bold(true))
				// 当显示 "ACCESS GRANTED" 时，它会清除屏幕。
				// 为确保滚动在临时消息后正确恢复，我们应清除内部缓冲区并重置键入位置。
				index = 0                     // 重置文本索引
				initScreenLines(screenHeight) // 清空屏幕缓冲区
				currentLine = 0
				currentCol = 0
				continue
			case tcell.KeyCtrlD: // Access Denied
				deniedStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack).Bold(true)
				showCentered(s, "ACCESS DENIED", deniedStyle)
				// 同 Ctrl+A，清除缓冲区并重置位置
				index = 0                     // 重置文本索引
				initScreenLines(screenHeight) // 清空屏幕缓冲区
				currentLine = 0
				currentCol = 0
				continue

			default:
				// 输出新字符段
				index = addCharactersToBuffer(index, speed) // 更新全局 'index'

				// 如果读取到 codeText 结尾，循环回去
				if index >= len(codeText) {
					index = 0 // 重置全局文本索引，从头开始
					// 清空屏幕缓冲区并重置光标，以便内容循环时从屏幕顶部重新开始
					initScreenLines(screenHeight)
					currentLine = 0
					currentCol = 0
				}
				redrawScreen(s, style)
			}
		}
	}
}
