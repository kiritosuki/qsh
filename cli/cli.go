package cli

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/kiritosuki/qsh/llm"
	"github.com/kiritosuki/qsh/utils"
)

type State int

const (
	Loading State = iota
	ReceivingInput
	ReceivingResponse
)

type model struct {
	client           *llm.LLMClient
	markdownRenderer *glamour.TermRenderer
	p                *tea.Program

	textInput textinput.Model
	spinner   spinner.Model

	state                 State
	query                 string
	latestCommandResponse string
	latestCommandIsCode   bool

	formattedPartialResponse string

	maxWidth int

	runWithArgs bool
	err         error
}

type responseMsg struct {
	response string
	err      error
}

type partialResponseMsg struct {
	content string
	err     error
}

type setPMsg struct {
	p *tea.Program
}

// makeQuery 调用AI查询
func makeQuery(client *llm.LLMClient, query string) tea.Cmd {
	return func() tea.Msg {
		response, err := client.Query(query)
		return responseMsg{response: response, err: err}
	}
}

// initialModel 初始化model
func initialModel(prompt string, client *llm.LLMClient) model {
	maxWidth := utils.GetTermSafeMaxWidth()
}

/* 给 model 实现 tea.Model 的三个接口 */

// Init 实现 tea.Model 的 Init 接口
func (m *model) Init() tea.Cmd {
	if m.runWithArgs {
		// 有参数 图标旋转 等待调用AI查询结果
		return tea.Batch(m.spinner.Tick, makeQuery(m.client, m.query))
	}
	// 有参数 光标闪烁 等待用户输入
	return textinput.Blink
}

// Update 实现 tea.Model 的 Update 接口
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	// 产生有效新消息 优先处理这些新消息
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc, tea.KeyCtrlD:
			return m, tea.Quit
		case tea.KeyEnter:
			return m.handleKeyEnter()
		}
	case responseMsg:
		return m.handleResponseMsg(msg)
	case partialResponseMsg:
		return m.handlePartialResponseMsg(msg)
	case setPMsg:
		m.p = msg.p
		return m, nil
	case error:
		m.err = msg
		return m, nil
	}
	// 产生无效新消息 就更新加载动画
	switch m.state {
	case Loading:
		// 等待AI处理提问 加载图标转圈
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case ReceivingInput:
		// 用户正在输入 处理文本/光标等
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
	return nil, cmd
}

// View 实现 tea.Model 的 View 接口
func (m *model) View() string {
	switch m.state {
	case Loading:
		// 等待AI处理提问 加载图标转圈
		return m.spinner.View()
	case ReceivingInput:
		// 等待用户输入 加载文本/光标等渲染
		return m.textInput.View()
	case ReceivingResponse:
		// AI返回新消息 刷新
		return m.formattedPartialResponse + "\n"
	}
	return ""
}

/* MsgHandlers */

func (m *model) handleKeyEnter() (tea.Model, tea.Cmd) {
	if m.state != ReceivingInput {
		return m, nil
	}
	// 获取输入框内容
	v := m.textInput.Value()
	// 输入框是空的 按下回车会进行代码复制 并退出
	if v == "" {
		if m.latestCommandResponse == "" {
			return m, tea.Quit
		}
		err := clipboard.WriteAll(m.latestCommandResponse)
		if err != nil {
			fmt.Println("Failed to copy text to clipboard:", err)
			return m, tea.Quit
		}
		// 把前景色渲染的暗一点
		placeholderStyle := lipgloss.NewStyle().Faint(true)
		message := "Copied to clipboard."
		if !m.latestCommandIsCode {
			message = "Only code can be copied to clipboard."
		}
		message = placeholderStyle.Render(message)
		return m, tea.Sequence(tea.Printf("%s", message), tea.Quit)
	}
	// 输入框有内容 提交问题给AI处理
	// 先把输入框清空 方便下次使用
	m.textInput.SetValue("")
	m.query = v
	m.state = Loading
	placeholderStyle := lipgloss.NewStyle().Faint(true).Width(m.maxWidth)
	message := placeholderStyle.Render(fmt.Sprintf("> %s", v))
	// 串行打印命令 然后执行后面的两个函数
	// Batch为并行执行 同时加载转圈动画与提交问题给AI
	return m, tea.Sequence(tea.Printf("%s", message), tea.Batch(m.spinner.Tick, makeQuery(m.client, m.query)))
}

func (m *model) handleResponseMsg(msg responseMsg) (tea.Model, tea.Cmd) {
	m.formattedPartialResponse = ""
	if msg.err != nil {
		m.state = ReceivingInput
		message := m.getConnectionError(msg.err)
		return m, tea.Sequence(tea.Printf("%s", message), textinput.Blink)
	}
	// 解析出代码块
	// content为纯净代码 isOnlyCode判断msg.response中是否有非代码内容
	content, isOnlyCode := utils.ExtractFirstCodeBlock(msg.response)
	if content != "" {
		m.latestCommandResponse = content
	}
	formatted, err := m.formatResponse(msg.response, utils.StartsWithCodeBlock(msg.response))
	if err != nil {
		panic(err)
	}
	m.textInput.Placeholder = "Done... Press ENTER to copy & quit, CTRL+C to quit."
	if !isOnlyCode {
		m.textInput.Placeholder = "Done... Press ENTER to copy the code, CTRL+C to quit."
	}
	if m.latestCommandResponse == "" {
		m.textInput.Placeholder = "Done... Press ENTER or CTRL+C to quit."
	}
	m.state = ReceivingInput
	m.latestCommandIsCode = isOnlyCode
	message := formatted
	return m, tea.Sequence(tea.Printf("%s", message), textinput.Blink)
}

func (m *model) handlePartialResponseMsg(msg partialResponseMsg) (tea.Model, tea.Cmd) {
	m.state = ReceivingResponse
	startsWithCode := utils.StartsWithCodeBlock(msg.content)
	formatted, err := m.formatResponse(msg.content, startsWithCode)
	if err != nil {
		panic(err)
	}
	m.formattedPartialResponse = formatted
	return m, nil
}

/* 辅助函数 */

func (m *model) getConnectionError(err error) string {
	styleRed := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleGreen := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	// 暗风格 暗色字体 + 宽度限制 + 左边距
	styleDim := lipgloss.NewStyle().Faint(true).Width(m.maxWidth).PaddingLeft(2)
	message := fmt.Sprintf("\n  %v\n\n%v\n",
		// TODO 后续把 AI 换成模型名称
		styleRed.Render("Error: Failed to connect to AI"),
		styleDim.Render(err.Error()),
	)
	if utils.IsLikelyBillingError(err.Error()) {
		message = fmt.Sprintf("%v\n  %v %v\n\n  %v%v\n\n",
			message,
			styleGreen.Render("Hint:"),
			"You may need to set up billing. You can do so here:",
			styleGreen.Render("->"),
			styleDim.Render("https://aihubmix.com"),
		)
	}
	return message
}

func (m *model) formatResponse(response string, startsWithCode bool) (string, error) {
	formatted, err := m.markdownRenderer.Render(response)
	if err != nil {
		return "", err
	}
	formatted = strings.TrimPrefix(formatted, "\n")
	formatted = strings.TrimSuffix(formatted, "\n")
	// 如果是以代码开头 将代码展示在">"的后面
	// 如果是以文字开头 增加换行符 来与用户输入的提示词分隔开
	if !startsWithCode {
		formatted = "\n" + formatted
	}
	return formatted, nil
}
