function apiURL(path) {
  return `https://api.github.com${path}`;
}

function encodePath(path) {
  return path.split("/").map((part) => encodeURIComponent(part)).join("/");
}

function headers(token, json = false) {
  return {
    Accept: "application/vnd.github+json",
    Authorization: `Bearer ${token}`,
    ...(json ? { "Content-Type": "application/json" } : {})
  };
}

async function request(token, path, options = {}) {
  const response = await fetch(apiURL(path), {
    ...options,
    headers: { ...headers(token, Boolean(options.body)), ...(options.headers || {}) }
  });
  const body = await response.text();
  let data;
  try {
    data = body ? JSON.parse(body) : null;
  } catch {
    data = body;
  }
  if (!response.ok) {
    const message = data?.message || response.statusText;
    throw new Error(`GitHub ${response.status}: ${message}`);
  }
  return data;
}

export async function getFile(token, owner, repo, branch, path) {
  try {
    return await request(token, `/repos/${owner}/${repo}/contents/${encodePath(path)}?ref=${encodeURIComponent(branch)}`);
  } catch (error) {
    if (error.message.startsWith("GitHub 404:")) return null;
    throw error;
  }
}

export async function putFile({ token, owner, repo, branch, path, content, message }) {
  for (let attempt = 0; attempt < 2; attempt++) {
    const current = await getFile(token, owner, repo, branch, path);
    const body = {
      message,
      content: btoa(unescape(encodeURIComponent(content))),
      branch
    };
    if (current?.sha) body.sha = current.sha;
    try {
      return await request(token, `/repos/${owner}/${repo}/contents/${encodePath(path)}`, {
        method: "PUT",
        body: JSON.stringify(body)
      });
    } catch (error) {
      if (attempt === 0 && error.message.includes("409")) {
        await new Promise((resolve) => setTimeout(resolve, 600));
        continue;
      }
      throw error;
    }
  }
}

export async function checkRepository(token, owner, repo) {
  return request(token, `/repos/${owner}/${repo}`);
}
