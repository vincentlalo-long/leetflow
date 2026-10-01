# LeetFlow Sync Extension

This directory contains the shared browser source for Chrome and Firefox. It archives accepted LeetCode submissions and reviews them later.

## First-time setup

1. Create or choose a GitHub repository for your solutions.
2. Create a GitHub token that can write to that repository. For a classic token, `repo` scope is required for private repositories; use the narrowest fine-grained repository permission that allows **Contents: Read and write** when possible.
3. From the repository root, build both browser packages:

   ```bash
   node build.mjs
   ```

4. Install the matching package:
   - Chrome: open `chrome://extensions`, enable **Developer mode**, choose **Load unpacked**, and select `dist/chrome/`.
   - Firefox (temporary, for development): open `about:debugging#/runtime/this-firefox`, choose **Load Temporary Add-on**, and select `dist/firefox/manifest.json`. Temporary add-ons are removed when Firefox closes.
   - Firefox (permanent): build `dist/leetflow-firefox.xpi`, open `about:addons`, click the gear icon and choose **Install Add-on From File…**. This requires an AMO-signed package; see the repository README for the signing workflow.

5. Open the LeetFlow extension menu and choose **Options**.
6. Fill in:
   - GitHub token
   - GitHub owner
   - Repository name
   - Branch, normally `main`
   - Root directory, normally `LeetCode`
7. Enable automatic sync and root README updates, then save.

The options page verifies that the repository is reachable before saving, but a repository that does not exist yet is allowed — the first sync creates it.

## Sync behaviour

- Every accepted submission is queued, then drained by a single background worker. Concurrent solutions are serialized and committed together.
- A solution, its problem `README.md`, and the root index are pushed as **one commit** through the GitHub Git Data API.
- Resubmitting an unchanged solution reports `unchanged` and does not create a commit; a modified solution creates a new commit.
- GitHub rate limits and transient errors leave the job in the queue and schedule a retry with backoff. Use the popup's **Sync / Retry now** button to force a run.
- The root `README.md` is merged inside `<!-- LEETFLOW:START -->` / `<!-- LEETFLOW:END -->` markers, so anything you write outside the block is never overwritten.

## Daily workflow

1. Open a LeetCode problem while logged in.
2. Solve it in the LeetCode editor.
3. Submit normally.
4. After LeetCode shows **Accepted**, leave the page open briefly so the content script can capture the accepted state.
5. Open the LeetFlow popup to see pending, failed, and synced records.
6. If GitHub was temporarily unavailable, press **Retry failed jobs**.

The extension never runs a second local judge. The official LeetCode result is the source of truth.

## Review workflow

Every accepted problem is added to the local review queue in extension storage.

1. Open the LeetFlow popup.
2. If a problem is due, read or revisit its solution.
3. Grade your recall:
   - **Again**: forgot the approach.
   - **Hard**: solved with substantial effort.
   - **Good**: solved normally.
   - **Easy**: recalled immediately.
4. The next due date is scheduled automatically.

The review data is local to the browser profile. Export/sync of review state is intentionally not part of the first extension MVP.

## Generated files

```text
LeetCode/
├── README.md
└── 0001-two-sum/
    ├── README.md
    └── two-sum.cpp
```

The root index has clickable topic anchors. Problem README files include the LeetCode link, difficulty badge, tags, accepted timestamp, solution link, and an editable notes section.

## Current limitations

- The unpacked MVP uses a manually configured GitHub token. Replace this with GitHub Device Flow before public distribution.
- LeetCode changes its DOM frequently. Accepted detection and editor extraction may need selector updates.
- The first version targets the common LeetCode Monaco editor path and may not capture every language/editor state.
- Images remain in the fetched problem description URL for now; downloading them into `assets/` is a later hardening step.

## Troubleshooting

**Nothing was queued after Accepted**

- Confirm the extension is enabled.
- Confirm the problem URL matches `/problems/...`.
- Leave the Accepted result visible for a moment.
- Reload the problem and submit again if the page changed before capture.

**Queue jobs are failed**

- Open Options and verify token, owner, repository, branch, and root directory.
- Confirm the token can write repository contents.
- Return to the popup and choose **Retry failed jobs**.

**Review list is empty**

- Only accepted submissions create review entries.
- Open the popup after a successful sync and check that the problem appears in the tracked count.
