# Leet CLI

Terminal tool to organize, solve, test, and submit LeetCode problems. Solutions are stored as plain files on disk, organized by data structure, and version-controlled with Git.

## Install

Requires [Go 1.25+](https://go.dev/dl/).

### Quick Install (Linux / macOS):
```bash
make install
# or: ./build.sh
```

### Windows (PowerShell):
```powershell
powershell -ExecutionPolicy Bypass -File build.ps1
# or: go build -o leet.exe .
```

## Quick Start

```bash
leet init               # interactive setup wizard (first-time)
leet add 1              # create file + README for Two Sum
leet open 1             # open workspace (auto tiling on Linux / Hyprland)
leet test 1 --local     # test locally (no cookies needed)
leet submit 1           # submit to LeetCode (needs cookies)
leet sync               # git commit + push
```

## Commands

| Command | Description |
|---------|-------------|
| `leet init` | First-run setup wizard |
| `leet add <num>` | Add problem from LeetCode |
| `leet daily` | Today's daily challenge |
| `leet random` | Random problem |
| `leet list` | List local problems |
| `leet search <q>` | Search by name/number |
| `leet open <num>` | Open in editor |
| `leet run <num>` | Compile & run |
| `leet test <num> --local` | Test with example cases |
| `leet test <num>` | Test via LeetCode API |
| `leet submit <num>` | Submit solution |
| `leet verify <file>` | Auto-test file (CI) |
| `leet review` | Spaced repetition queue |
| `leet stats` | Workspace statistics |
| `leet doctor` | Health check |
| `leet readme` | Generate index |
| `leet sync` | Commit & push |
| `leet completion bash` | Shell completion |

## Configuration

Config file: `config.json` (committed) + `config.local.json` (git-ignored, for secrets).

Key settings:

| Setting | Key | Env var |
|---------|-----|---------|
| Workspace directory | `base_dir` | — |
| Default language | `default_language` | — |
| Editor | `editor` | — |
| LeetCode session | `leetcode_session` | `LEETCODE_SESSION` |
| LeetCode CSRF | `leetcode_csrf` | `LEETCODE_CSRF` |

Run `leet config` to edit interactively.

## Workspace Structure

```
<base_dir>/
├── array/
│   └── 1-two-sum/
│       ├── 1_two_sum.cpp
│       └── README.md
├── tree/
├── dp/
└── ...
```

## Tests

```bash
cd go-cli
go test ./...
```

## License

MIT
