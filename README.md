# LeetFlow Sync

A browser extension that syncs accepted LeetCode solutions to a GitHub repository and provides a local review queue.

## Features

- **Accepted-only sync**: Automatically captures accepted submissions from LeetCode.com and LeetCode.cn.
- **Repository layout**: Saves solutions into numbered folders (`0001-two-sum/`) with proper file extensions and a generated problem `README.md`.
- **Root catalog**: Maintains an index `README.md` organized by problem number and topic tags, wrapped in a `<!-- LEETFLOW:START/END -->` marker so anything you write around it is preserved.
- **One commit per sync**: solution file, problem README, and root index are pushed in a single commit through the GitHub Git Data API.
- **Resilient sync**: rate limits, moved branches, and transient GitHub errors are retried with backoff; failed jobs stay in the queue until they succeed.
- **Quick notes prompt**: Optional toast prompt when a submission is accepted to jot down key insights or tricks.
- **Multi-Approach**: Label each submission's approach (e.g. Brute Force, HashMap) in the notes prompt — every approach is archived as its own file (`two-sum-hashmap.cpp`) and listed in the problem README instead of overwriting the previous one.
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

1. Download `leetflow-chrome-<tag>.zip` from [Releases](https://github.com/vincentlalo-long/leetflow/releases).
2. Extract the `.zip` file into a folder.
3. In Chrome, open `chrome://extensions`.
4. Enable **Developer mode** (top-right toggle).
5. Click **Load unpacked** and select the extracted folder.

**Firefox (permanent install)**

Temporary add-ons loaded through `about:debugging` disappear when Firefox closes. Use the AMO-signed `.xpi` instead:

1. Download `leetflow-firefox-<tag>.xpi` from [Releases](https://github.com/vincentlalo-long/leetflow/releases).
2. Open `about:addons`.
3. Click the gear icon → **Install Add-on From File…** and choose the `.xpi`.
4. Confirm the installation. The add-on now survives Firefox restarts.

No Firefox account and no AMO login are required to install a signed `.xpi`. Firefox shows a "not verified" prompt for unlisted add-ons — that is expected, click **Continue**.

Every release carries exactly two assets: `leetflow-chrome-<tag>.zip` (load unpacked) and `leetflow-firefox-<tag>.xpi` (already signed by Mozilla).

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

> **End users can ignore this section** — the signed `.xpi` is already attached to every Release. You only need AMO credentials if you maintain the extension and want to publish a new signed version.

`about:debugging → Load Temporary Add-on` is temporary by design. To distribute the extension permanently you need an AMO-signed `.xpi`:

1. Create a Firefox account and sign in to <https://addons.mozilla.org>.
2. Accept the **Firefox Add-on Distribution Agreement** (the API key page stays locked until you do).
3. Open the API key page: <https://addons.mozilla.org/developers/addon/api/key/> and generate a credential pair.
   - **JWT issuer** (API key), looks like `user:12345678:987`.
   - **JWT secret** (API secret), 64 hex characters — **shown only once**; if you lose it, generate a new pair (the old one is revoked).
4. Add two repository secrets to this GitHub repository (`Settings → Secrets and variables → Actions → New repository secret`):
   - `AMO_JWT_ISSUER` — the JWT issuer.
   - `AMO_JWT_SECRET` — the JWT secret.
5. Publish a release:
   ```bash
   npm version 0.7.1 --no-git-tag-version   # AMO rejects a reused version
   git commit -am "release: v0.7.1"
   git push
   git tag v0.7.1 && git push origin v0.7.1
   ```
   The **Release** workflow runs the tests, builds the packages, publishes the Chrome `.zip`, then calls the **Sign Firefox Add-on** job which overwrites the Release's Firefox asset with the signed `.xpi`.

   To sign without publishing (e.g. a test run), trigger **Sign Firefox Add-on** manually from the Actions tab — the `.xpi` is then available under that run's **Artifacts**.

You do **not** need to publish the add-on publicly — the `unlisted` channel keeps it out of the store while still allowing normal installation. The credentials are account-wide and do not expire.

## Development Checks

```bash
npm test          # unit + integration tests
node build.mjs    # builds dist/chrome, dist/firefox, .zip and .xpi
npx web-ext lint --source-dir dist/firefox
npm run check     # build + tests
```

### Test a build before releasing

```bash
node build.mjs
```

- **Chrome**: `chrome://extensions` → enable **Developer mode** → **Load unpacked** → `dist/chrome/`
- **Firefox**: `about:debugging#/runtime/this-firefox` → **Load Temporary Add-on** → `dist/firefox/manifest.json`

Temporary add-ons disappear when Firefox closes — expected while developing. Neither option consumes an AMO version.

### Releasing

Every tag is signed once by AMO and **AMO rejects a reused version number**, so tag only when the build is ready:

```bash
npm version 0.7.2 --no-git-tag-version
git commit -am "release: v0.7.2"
git push
git tag v0.7.2 && git push origin v0.7.2
```

The Release workflow runs the tests, builds, publishes `leetflow-chrome-<tag>.zip`, then signs and attaches `leetflow-firefox-<tag>.xpi`.

## Security Note

This developer version stores the GitHub token locally in `chrome.storage.local`. Do not commit extension storage exports or share tokens.

## License

MIT License. See [LICENSE](LICENSE).
