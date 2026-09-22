# leetflow

[![CI](https://github.com/vincentlalo-long/leetflow/actions/workflows/ci.yml/badge.svg)](https://github.com/vincentlalo-long/leetflow/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)

**A terminal LeetCode companion: solve locally, test offline, and actually remember solutions with spaced repetition.**

Solutions are stored as plain source files on disk, organized by data structure, and version-controlled with Git.

---

## ⚡ Features

- 🖥️ **Interactive Terminal UI**: Fast, keyboard-driven interface built with Bubble Tea.
- 🧪 **Offline Local Testing**: Test solutions against example test cases without internet or cookies (supports C++, Python, Java, Go).
- 🧠 **Spaced Repetition Review**: Smart review intervals (1, 3, 7, 15, 30, 60 days) to retain algorithmic patterns.
- 📂 **Structured Workspace**: Solutions auto-sorted into topic directories (`array/`, `tree/`, `dp/`, `graph/`).
- 🔄 **Git Integration**: Auto-commit progress and push to your solutions repository with `leet sync`.
- 🌐 **LeetCode Integration**: Fetch daily challenges, submit solutions, and track submission metrics directly from terminal.

---

## 📦 Installation

Requires [Go 1.25+](https://go.dev/dl/).

### Pre-built Binaries (GitHub Releases)
Download the latest pre-built binary for Linux, macOS, or Windows from the [Releases](https://github.com/vincentlalo-long/leetflow/releases) page.

### Build from Source

#### Linux / macOS:
```bash
git clone https://github.com/vincentlalo-long/leetflow.git
cd leetflow
make install
# or: ./build.sh
```

#### Windows (PowerShell):
```powershell
git clone https://github.com/vincentlalo-long/leetflow.git
cd leetflow
powershell -ExecutionPolicy Bypass -File build.ps1
# or: go build -o leet.exe .
```

---

## 🚀 Quick Start

```bash
# 1. Interactive setup wizard
leet init

# 2. Add problem #1 (Two Sum) with template & README
leet add 1

# 3. Open in your preferred editor
leet open 1

# 4. Test locally with example test cases (offline, no cookies needed)
leet test 1 --local

# 5. Submit directly to LeetCode (requires session cookie)
leet submit 1

# 6. Check your spaced repetition review queue
leet review

# 7. Commit & push solutions to your git repository
leet sync
```

---

## 🛠️ Commands Reference

| Command | Description |
|---------|-------------|
| `leet` | Launch interactive Terminal UI (TUI) |
| `leet init` | First-run interactive setup wizard |
| `leet add <num>` | Fetch & scaffold problem from LeetCode |
| `leet daily` | Fetch today's Daily Challenge |
| `leet random` | Pick a random problem (filter by difficulty/tag) |
| `leet list` | List all local problems with solving status |
| `leet search <query>` | Search local problems by name or number |
| `leet open <num>` | Open problem in your editor |
| `leet run <num>` | Compile & run local solution |
| `leet test <num> --local` | Run offline local test harness |
| `leet test <num>` | Run remote test on LeetCode API |
| `leet submit <num>` | Submit solution to LeetCode |
| `leet verify <file>` | Headless test runner (useful for CI) |
| `leet review` | Review problems due in spaced repetition queue |
| `leet stats` | Workspace stats, solve rates & category breakdown |
| `leet doctor` | System health check (compiler, git, credentials) |
| `leet readme` | Re-generate catalog README.md of solved problems |
| `leet sync` | Auto-commit and git push solutions |
| `leet completion <shell>` | Generate shell completions (bash, zsh, fish) |

---

## ⚙️ Configuration

Settings are managed via `config.json` (committed) and `config.local.json` (git-ignored, saved with `0600` permissions for credential protection).

| Setting | Key | Env var | Description |
|---------|-----|---------|-------------|
| Workspace directory | `base_dir` | — | Root directory for problem files |
| Default language | `default_language` | — | Default language (`cpp`, `python3`, `golang`, `java`) |
| Editor command | `editor` | — | Editor to launch (`nvim`, `code`, `vim`) |
| LeetCode session | `leetcode_session` | `LEETCODE_SESSION` | `LEETCODE_SESSION` cookie from browser |
| LeetCode CSRF token | `leetcode_csrf` | `LEETCODE_CSRF` | `csrftoken` cookie from browser |

Run `leet config` to view or edit configuration interactively.

---

## 🧪 Testing

Run all unit tests:
```bash
go test -v ./...
```

Run tests with data race detector:
```bash
go test -race ./...
```

---

## 📄 License

[MIT License](LICENSE) © 2026 vincentlalo-long
