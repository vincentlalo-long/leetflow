# LeetFlow Sync

A browser extension that syncs accepted LeetCode solutions to a GitHub repository and provides a local review queue.

## Features

- **Accepted-only sync**: Automatically captures accepted submissions from LeetCode.com and LeetCode.cn.
- **Repository layout**: Saves solutions into numbered folders (`0001-two-sum/`) with proper file extensions and a generated problem `README.md`.
- **Root catalog**: Maintains an index `README.md` organized by problem number and topic tags.
- **Quick notes prompt**: Optional toast prompt when a submission is accepted to jot down key insights or tricks.
- **Solve telemetry**: Records elapsed time and submission attempts (WA/TLE/AC) for each problem.
- **Review queue**: Spaced repetition review scheduling in the extension popup with Again, Hard, Good, and Easy grades.
- **Roadmap tracking**: Basic progress tracking for Blind 75 and NeetCode 150 problem lists.
- **Interview mode**: Optional toggle to blur topic tags on problem statements to prevent spoilers while practicing.
- **Export to Anki**: Export synced problems to a TSV file for import into Anki flashcards.

## Workflow

```text
LeetCode Accepted
       ↓
Content script captures code & metadata
       ↓
GitHub Contents API commit (Solution + README)
       ↓
Local review queue in extension popup
```

## Generated Repository Structure

```text
LeetCode/
├── README.md
└── 0001-two-sum/
    ├── README.md
    └── two-sum.cpp
```

## Installation and Setup

### 1. Build

Ensure Node.js is installed:

```bash
node build.mjs
```

This outputs ready-to-load extension packages into `dist/chrome/` and `dist/firefox/`.

### 2. Load Extension

- **Chrome / Chromium**:
  1. Go to `chrome://extensions`.
  2. Enable **Developer mode** (top-right).
  3. Click **Load unpacked** and select the `dist/chrome/` folder.

- **Firefox**:
  1. Go to `about:debugging#/runtime/this-firefox`.
  2. Click **Load Temporary Add-on** and select `dist/firefox/manifest.json`.

### 3. Configuration

1. Open the extension popup and click **Settings** (or right-click the extension icon -> Options).
2. Enter:
   - **GitHub token**: Personal access token with repository write permissions.
   - **Owner**: Your GitHub username or organization.
   - **Repository**: The repository name (e.g. `leetcode-solutions`).
   - **Branch**: Target branch (default `main`).
   - **Root directory**: Target directory in repo (default `LeetCode`).
3. Click **Save settings**.

## Development Checks

Run syntax validation and smoke checks:

```bash
for f in extension/*.js; do node --check "$f"; done
node --input-type=module <<'EOF'
import assert from "node:assert/strict";
import { renderRootReadme } from "./extension/templates.js";
assert.match(renderRootReadme([], { rootDir: "LeetCode" }), /LeetCode Solutions/);
console.log("smoke test passed");
EOF
```

## Security Note

This developer version stores the GitHub token locally in `chrome.storage.local`. Do not commit extension storage exports or share tokens.

## License

MIT License. See [LICENSE](LICENSE).
