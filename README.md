# LeetFlow Sync

A browser extension that syncs accepted LeetCode solutions to a GitHub repository and provides a local review queue.

## Features

- **Accepted-only sync**: Automatically captures accepted submissions from LeetCode.com and LeetCode.cn.
- **Repository layout**: Saves solutions into numbered folders (`0001-two-sum/`) with proper file extensions and a generated problem `README.md`.
- **Root catalog**: Maintains an index `README.md` organized by problem number and topic tags, wrapped in a `<!-- LEETFLOW:START/END -->` marker so anything you write around it is preserved.
- **One commit per sync**: solution file, problem README, and root index are pushed in a single commit through the GitHub Git Data API.
- **Resilient sync**: rate limits, moved branches, and transient GitHub errors are retried with backoff; failed jobs stay in the queue until they succeed.
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
Queue job (mutex-protected, retried with backoff)
       ↓
GitHub Git Data API: 1 commit (Solution + Problem README + Root README)
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

## Installation

### Option 1: Pre-built Package (Recommended)

No Node.js or terminal required:

**Chrome / Edge**

1. Download `leetflow-chrome-v0.7.0.zip` from [Releases](https://github.com/vincentlalo-long/leetflow/releases).
2. Extract the `.zip` file into a folder.
3. In Chrome, open `chrome://extensions`.
4. Enable **Developer mode** (top-right toggle).
5. Click **Load unpacked** and select the extracted folder.

**Firefox (permanent install)**

Temporary add-ons loaded through `about:debugging` disappear when Firefox closes. Use the AMO-signed `.xpi` instead:

1. Download `leetflow-firefox-v0.7.0.xpi` from [Releases](https://github.com/vincentlalo-long/leetflow/releases).
2. Open `about:addons`.
3. Click the gear icon → **Install Add-on From File…** and choose the `.xpi`.
4. Confirm the installation. The add-on now survives Firefox restarts.

### Option 2: Build from Source

```bash
git clone https://github.com/vincentlalo-long/leetflow.git
cd leetflow
node build.mjs
```

Load the generated `dist/chrome/` folder via **Load unpacked**, or install `dist/leetflow-firefox.xpi` as described above.

### 3. Configuration

1. Open the extension popup and click **Settings** (or right-click the extension icon -> Options).
2. Enter:
   - **GitHub token**: Personal access token with repository write permissions.
   - **Owner**: Your GitHub username or organization.
   - **Repository**: The repository name (e.g. `leetcode-solutions`).
   - **Branch**: Target branch (default `main`).
   - **Root directory**: Target directory in repo (default `LeetCode`).
3. Click **Save settings**.

## Signing the Firefox Add-on

`about:debugging → Load Temporary Add-on` is temporary by design. To install the extension permanently you need an AMO-signed `.xpi`:

1. Create a Firefox account and sign in to <https://addons.mozilla.org>.
2. Accept the **Firefox Add-on Distribution Agreement** (the API key page stays locked until you do).
3. Open the API key page: <https://addons.mozilla.org/developers/addon/api/key/> and generate a credential pair.
   - **JWT issuer** (API key), looks like `user:12345678:987`.
   - **JWT secret** (API secret), 64 hex characters — **shown only once**; if you lose it, generate a new pair (the old one is revoked).
4. Add two repository secrets to this GitHub repository (`Settings → Secrets and variables → Actions → New repository secret`):
   - `AMO_JWT_ISSUER` — the JWT issuer.
   - `AMO_JWT_SECRET` — the JWT secret.
5. Run the **Sign Firefox Add-on** workflow from the **Actions** tab (`Run workflow`), or publish a GitHub release — the workflow also runs on `release: published`.
6. Download the signed `.xpi` from the workflow run's **Artifacts** (`leetflow-firefox-signed`), or from the release assets.

You do **not** need to publish the add-on publicly — the `unlisted` channel keeps it out of the store while still allowing normal installation. The credentials are account-wide and do not expire.

## Development Checks

```bash
npm test          # unit + integration tests
node build.mjs    # builds dist/chrome, dist/firefox, .zip and .xpi
npx web-ext lint --source-dir dist/firefox
npm run check     # build + tests
```

## Security Note

This developer version stores the GitHub token locally in `chrome.storage.local`. Do not commit extension storage exports or share tokens.

## License

MIT License. See [LICENSE](LICENSE).
