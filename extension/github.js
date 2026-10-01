const API_ROOT = "https://api.github.com";
const API_VERSION = "2022-11-28";

export class GitHubError extends Error {
  constructor(message, options = {}) {
    super(message);
    this.name = "GitHubError";
    this.status = options.status || 0;
    this.rateLimited = Boolean(options.rateLimited);
    this.resetAt = options.resetAt || 0;
    this.data = options.data || null;
  }
}

function apiURL(path) {
  return `${API_ROOT}${path}`;
}

function encodePath(path) {
  return String(path)
    .split("/")
    .map((part) => encodeURIComponent(part))
    .join("/");
}

function encodeRef(branch) {
  return String(branch)
    .split("/")
    .map((part) => encodeURIComponent(part))
    .join("/");
}

function buildHeaders(token, json) {
  return {
    Accept: "application/vnd.github+json",
    "X-GitHub-Api-Version": API_VERSION,
    Authorization: `Bearer ${token}`,
    ...(json ? { "Content-Type": "application/json" } : {})
  };
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function readRateLimit(response) {
  const remaining = response.headers.get("x-ratelimit-remaining");
  const resetSeconds = Number(response.headers.get("x-ratelimit-reset")) || 0;
  const retryAfter = Number(response.headers.get("retry-after")) || 0;
  const resetAt = retryAfter
    ? Date.now() + retryAfter * 1000
    : resetSeconds
      ? resetSeconds * 1000
      : Date.now() + 60000;
  return { remaining, resetAt };
}

async function request(token, path, options = {}) {
  if (!token) {
    throw new GitHubError("Missing GitHub token. Add one in Settings.");
  }
  const response = await fetch(apiURL(path), {
    ...options,
    headers: {
      ...buildHeaders(token, Boolean(options.body)),
      ...(options.headers || {})
    }
  });

  const raw = await response.text();
  let data = null;
  if (raw) {
    try {
      data = JSON.parse(raw);
    } catch {
      data = raw;
    }
  }

  const { remaining, resetAt } = readRateLimit(response);
  const secondaryLimit =
    typeof data === "object" &&
    data &&
    /rate limit|abuse detection/i.test(String(data.message || ""));

  if ((response.status === 403 && (remaining === "0" || secondaryLimit)) || response.status === 429) {
    const message =
      typeof data === "object" && data?.message ? data.message : "GitHub API rate limit exceeded.";
    throw new GitHubError(`GitHub rate limit: ${message}`, {
      status: response.status,
      rateLimited: true,
      resetAt,
      data
    });
  }

  if (!response.ok) {
    const message =
      typeof data === "object" && data?.message ? data.message : response.statusText;
    throw new GitHubError(`GitHub ${response.status}: ${message}`, {
      status: response.status,
      data
    });
  }

  return data;
}

function isConflict(error) {
  if (!(error instanceof GitHubError)) return false;
  if (error.rateLimited) return false;
  return error.status === 409 || error.status === 422;
}

export function decodeBase64(value) {
  if (!value) return "";
  const binary =
    typeof atob === "function"
      ? atob(value)
      : Buffer.from(value, "base64").toString("binary");
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
  return new TextDecoder().decode(bytes);
}

export async function getRefHead(token, owner, repo, branch = "main") {
  try {
    const ref = await request(
      token,
      `/repos/${owner}/${repo}/git/ref/heads/${encodeRef(branch)}`
    );
    return ref?.object?.sha || null;
  } catch (error) {
    if (error.status === 404) return null;
    throw error;
  }
}

export async function getCommit(token, owner, repo, sha) {
  return request(token, `/repos/${owner}/${repo}/git/commits/${encodeURIComponent(sha)}`);
}

export async function createTree(token, owner, repo, { baseTree, entries }) {
  const tree = entries.map((entry) => ({
    path: entry.path,
    mode: "100644",
    type: "blob",
    content: String(entry.content ?? "")
  }));
  const body = { tree };
  if (baseTree) body.base_tree = baseTree;
  return request(token, `/repos/${owner}/${repo}/git/trees`, {
    method: "POST",
    body: JSON.stringify(body)
  });
}

export async function createCommit(token, owner, repo, { message, tree, parents }) {
  return request(token, `/repos/${owner}/${repo}/git/commits`, {
    method: "POST",
    body: JSON.stringify({ message, tree, parents })
  });
}

export async function updateRef(token, owner, repo, branch, sha) {
  return request(token, `/repos/${owner}/${repo}/git/refs/heads/${encodeRef(branch)}`, {
    method: "PATCH",
    body: JSON.stringify({ sha, force: false })
  });
}

export async function createRef(token, owner, repo, branch, sha) {
  return request(token, `/repos/${owner}/${repo}/git/refs`, {
    method: "POST",
    body: JSON.stringify({ ref: `refs/heads/${branch}`, sha })
  });
}

export async function getFile(token, owner, repo, branch, path) {
  if (!token) return null;
  try {
    const query = branch ? `?ref=${encodeURIComponent(branch)}` : "";
    return await request(
      token,
      `/repos/${owner}/${repo}/contents/${encodePath(path)}${query}`
    );
  } catch (error) {
    if (error.status === 404) return null;
    throw error;
  }
}

export async function readRemoteFile(token, owner, repo, branch, path) {
  const entry = await getFile(token, owner, repo, branch, path);
  if (!entry || typeof entry.content !== "string" || entry.encoding !== "base64") {
    return "";
  }
  return decodeBase64(entry.content);
}

export async function checkRepository(token, owner, repo) {
  if (!token) throw new GitHubError("Missing GitHub token.");
  return request(token, `/repos/${owner}/${repo}`);
}

export async function commitFiles({ token, owner, repo, branch = "main", files, message }) {
  if (!owner || !repo) {
    throw new GitHubError("Configure the GitHub owner and repository in Settings first.");
  }
  const unique = new Map();
  for (const file of files || []) {
    if (file?.path) unique.set(file.path, { path: file.path, content: String(file.content ?? "") });
  }
  const entries = [...unique.values()];
  if (entries.length === 0) {
    throw new GitHubError("Nothing to commit.");
  }

  let lastError = null;
  for (let attempt = 0; attempt < 3; attempt++) {
    try {
      const head = await getRefHead(token, owner, repo, branch);
      let baseTree = null;
      let parents = [];
      if (head) {
        const commit = await getCommit(token, owner, repo, head);
        baseTree = commit?.tree?.sha || null;
        parents = [head];
      }
      const tree = await createTree(token, owner, repo, { baseTree, entries });
      const commit = await createCommit(token, owner, repo, {
        message,
        tree: tree.sha,
        parents
      });
      if (head) {
        await updateRef(token, owner, repo, branch, commit.sha);
      } else {
        await createRef(token, owner, repo, branch, commit.sha);
      }
      return { sha: commit.sha, files: entries.length, branch };
    } catch (error) {
      lastError = error;
      if (error.rateLimited) throw error;
      if (attempt < 2 && isConflict(error)) {
        await sleep(300 * (attempt + 1));
        continue;
      }
      throw error;
    }
  }
  throw lastError || new GitHubError("Commit failed.");
}
