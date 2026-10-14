export const UPDATE_REPO = "vincentlalo-long/leetflow";
export const RELEASES_LATEST_URL = `https://api.github.com/repos/${UPDATE_REPO}/releases/latest`;
export const RELEASES_PAGE_URL = `https://github.com/${UPDATE_REPO}/releases`;

export function getExtensionVersion() {
  try {
    const manifest = globalThis.chrome?.runtime?.getManifest?.();
    if (manifest?.version) return String(manifest.version);
  } catch {
    /* fall through to the fallback below */
  }
  return "0.0.0";
}

export function normalizeVersion(value) {
  return String(value ?? "").trim().replace(/^v/i, "");
}

export function parseVersion(value) {
  const clean = normalizeVersion(value).split(/[+-]/)[0];
  const parts = clean.split(".").map((part) => {
    const num = Number(part);
    return Number.isInteger(num) && num >= 0 ? num : 0;
  });
  while (parts.length < 3) parts.push(0);
  return parts.slice(0, 3);
}

export function compareVersions(a, b) {
  const left = parseVersion(a);
  const right = parseVersion(b);
  for (let i = 0; i < 3; i++) {
    if (left[i] > right[i]) return 1;
    if (left[i] < right[i]) return -1;
  }
  return 0;
}

export function isUpdateAvailable(current, latest) {
  if (!normalizeVersion(latest)) return false;
  return compareVersions(normalizeVersion(latest), normalizeVersion(current)) > 0;
}

export async function fetchLatestRelease(fetchImpl = globalThis.fetch) {
  if (typeof fetchImpl !== "function") {
    throw new Error("Network is unavailable — could not check for updates.");
  }
  let response;
  try {
    response = await fetchImpl(RELEASES_LATEST_URL, {
      headers: { Accept: "application/vnd.github+json" }
    });
  } catch {
    throw new Error("Network error — could not check for updates. Try again later.");
  }
  if (response?.status === 404) {
    throw new Error("No published releases found yet.");
  }
  if (response?.status === 403) {
    throw new Error("GitHub rate limit reached — try checking again in a few minutes.");
  }
  if (!response?.ok) {
    throw new Error(`Could not check for updates (HTTP ${response?.status || "?"}).`);
  }
  const payload = await response.json();
  const version = normalizeVersion(payload?.tag_name || payload?.name || "");
  if (!version) {
    throw new Error("The latest release has no version tag.");
  }
  return {
    version,
    url: payload?.html_url || RELEASES_PAGE_URL,
    notes: String(payload?.body || "").slice(0, 2000),
    publishedAt: payload?.published_at || ""
  };
}

export async function checkForUpdate(current, fetchImpl) {
  const latest = await fetchLatestRelease(fetchImpl);
  return {
    current: normalizeVersion(current),
    latest: latest.version,
    updateAvailable: isUpdateAvailable(current, latest.version),
    url: latest.url,
    publishedAt: latest.publishedAt
  };
}
