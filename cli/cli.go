package cli

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/kiritosuki/qsh/config"
	"github.com/kiritosuki/qsh/llm"
	. "github.com/kiritosuki/qsh/types"
	"github.com/kiritosuki/qsh/utils"
	"github.com/spf13/cobra"
)

type State int

const (
	Loading           State = iota // 发送了请求给AI，但还没收到回复
	ReceivingInput                 // 等待用户输入
	ReceivingResponse              // AI 正在不断生成回复(已有生成)，流式接收中
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
func initialModel(prompt string, client *llm.LLMClient) *model {
	maxWidth := utils.GetTermSafeMaxWidth()
	ti := textinput.New()
	ti.Placeholder = "Describe a shell command, or ask a question."
	ti.Focus()
	ti.Width = maxWidth

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	runWithArgs := prompt != ""

	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(int(maxWidth)),
	)

	model := model{
		client:           client,
		markdownRenderer: r,
		textInput:        ti,
		spinner:          s,
		state:            ReceivingInput,
		maxWidth:         maxWidth,
		err:              nil,
	}
	if runWithArgs {
		model.runWithArgs = true
		model.state = Loading
		model.query = prompt
	}
	return &model
}

/* 给 model 实现 tea.Model 的三个接口 */

// model 有三个接口：Init Update View
// 由于执行了p.Run() 这三个方法会接替调用
// 先调用Init一次 之后调用Update占用前台线程阻塞 Update监听到Msg后会调用View来更新渲染
// tea.Cmd 是一个函数：func() Msg
// Init 会返回 tea.Cmd，这个 Cmd 会被交给框架放在后台开启新协程执行 返回的 Msg 会交给 Update
// Update 会阻塞监听返回的 Msg，来源有：
// 1. 键盘事件
// 2. Init / Update 返回的 Cmd 在后台执行，返回的 Msg
// 3. p.Send
// 当 Update 执行完更新逻辑后 框架自动调用 View 进行重新渲染
// 例如 spinner 没有调用 p.Run() 可以手动调用 Update View 方法

// Init 实现 tea.Model 的 Init 接口
func (m *model) Init() tea.Cmd {
	if m.runWithArgs {
		// 有参数 图标旋转 等待调用AI查询结果
		return tea.Batch(m.spinner.Tick, makeQuery(m.client, m.query))
	}
	// 无参数 光标闪烁 等待用户输入
	return textinput.Blink
}

// Update 实现 tea.Model 的 Update 接口
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	// 优先处理这些Msg
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
	// 如果没有收到上面的Msg 处理这些动画更新Msg
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
	return m, nil
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
			message = "Only the code was copied to clipboard."
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
	// 如果是以文字开头 增加换行符 来与用户输入的提示词分隔开
	if !startsWithCode {
		formatted = "\n" + formatted
	}
	return formatted, nil
}

func NewStreamCallback(p *tea.Program) func(content string, err error) {
	return func(content string, err error) {
		p.Send(partialResponseMsg{content, err})
	}
}

func printAPIKeyNotSetMessage(modelConfig ModelConfig) {
	auth := modelConfig.Auth
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
	)

	profileScriptName := ".zshrc or.bashrc"
	shellSyntax := "\n```bash\nexport QSH_API_KEY=[your key]\n```"
	if runtime.GOOS == "windows" {
		profileScriptName = "$profile"
		shellSyntax = "\n```powershell\n$env:QSH_API_KEY = \"[your key]\"\n```"
	}

	styleRed := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	switch auth {
	case "QSH_API_KEY":
		msg1 := styleRed.Render("QSH_API_KEY environment variable not set.")

		// make it platform agnostic
		message_string := fmt.Sprintf(`
	1. Generate your API key at https://aihubmix.com/
	2. Add your credit card in the API (for the free trial)
	3. Set your key by running:
	%s
	4. (Recommended) Add that ^ line to your %s file.`, shellSyntax, profileScriptName)

		msg2, _ := r.Render(message_string)
		fmt.Printf("\n  %v%v\n", msg1, msg2)
	default:
		msg := styleRed.Render(auth + " environment variable not set.")
		fmt.Printf("\n  %v", msg)
	}
}

/* Main */

func runQProgram(prompt string) {
	// 加载已有配置或者创建默认配置
	appConfig, err := config.LoadAppConfig()
	if err != nil {
		config.PrintConfigErrorMessage(err)
		os.Exit(1)
	}

	modelConfig, err := config.GetModelConfig(appConfig)
	if err != nil {
		config.PrintConfigErrorMessage(err)
		os.Exit(1)
	}
	auth := os.Getenv(modelConfig.Auth)
	if auth == "" {
		printAPIKeyNotSetMessage(modelConfig)
		os.Exit(1)
	}

	orgID := os.Getenv(modelConfig.OrgID)
	modelConfig.Auth = auth
	modelConfig.OrgID = orgID

	c := llm.NewLLMClient(modelConfig)
	p := tea.NewProgram(initialModel(prompt, c))
	c.StreamCallback = NewStreamCallback(p)
	if _, err := p.Run(); err != nil {
		fmt.Printf("OOPS, there's been an error: %v", err)
		os.Exit(1)
	}
}

var RootCmd = &cobra.Command{
	Use:   "q [request]",
	Short: "A command line interface for natural language queries",
	Run: func(cmd *cobra.Command, args []string) {
		// 把所有参数合并成一个字符串 用空格分隔
		prompt := strings.Join((args), " ")
		if len(args) > 0 && args[0] == "config" {
			config.RunConfigProgram(args)
			return
		}
		runQProgram(prompt)
	},
}
