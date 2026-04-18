# qsh 🐚

qsh 是一个轻量级的命令行 AI 助手，让你在终端中通过自然语言直接获取技术帮助。

> 致敬： 本项目参考了 https://github.com/ibigio/shell-ai 的优秀设计思路与实现，仅用于我个人学习。

---

## 1. 🚀 安装 (Installation)

### MacOs / Linux

- 通过 `curl` 安装，默认安装 `latest` 版本：

```bash
curl -fsSL https://raw.githubusercontent.com/kiritosuki/qsh/main/install.sh | bash
```

- 安装指定版本：

```bash
curl -fsSL https://raw.githubusercontent.com/kiritosuki/qsh/main/install.sh | env VERSION=v0.1.0 bash
```

安装完成后，你可以直接使用 `q`  命令。

### Windows

暂未提供直接安装脚本，可以选择已发布的版本手动下载。

------

## 2. ⚡ 快速开始 (Quick Start)

### 2-1. 配置 API Key

qsh 需要接入 AI 大模型，请先获取你的 API Key：

访问 **AIHubMix 控制台** 获取 Key。

设置环境变量：

- 临时生效：

```
export QSH_API_KEY=你的_API_KEY
```

- 持久化生效（推荐）：

根据你使用的 Shell，将上述命令添加到配置文件中：

```
# Zsh 用户
echo 'export QSH_API_KEY=你的_API_KEY' >> ~/.zshrc && source ~/.zshrc

# Bash 用户
echo 'export QSH_API_KEY=你的_API_KEY' >> ~/.bashrc && source ~/.bashrc
```

### 2-2. 开始提问

直接在 `q` 后面跟上你的问题即可：

```
q "如何递归删除当前目录下所有的 .log 文件？"
q "用 docker-compose 起一个 nginx 服务"
```

------

## 3. 🛠️ 高级配置 (Advanced Configuration)

qsh 提供了极高的自由度，你可以通过交互式命令或直接编辑配置文件来微调它的行为。

### 3-1. 交互式配置

输入以下命令进入交互式设置界面，快速切换模型：

```
q config
```

### 3-2. 手动编辑配置文件

配置文件通常位于：

```
~/.config/qsh/config.yaml
```

（或程序提示的其他路径）

你可以通过编辑该文件实现以下功能：

- 修改 Endpoint：对接不同的 API 中转站
- 更换模型：指定 `gpt-4o`、`claude-3-5-sonnet` 或其他兼容 OpenAI 接口的模型
- 自定义提示词（System Prompt）：修改 AI 的“性格”或预设背景
- 增加模型列表：在配置文件中预设多个常用模型以便快速切换

------

## 📜 许可证 (License)

本项目遵循仓库中提供的开源协议 [MIT License](LICENSE)

------

## 💡 提示

如果你在安装或使用过程中遇到任何问题，欢迎提交 Issue。
