export const DEFAULT_SETTINGS = {
  githubToken: "",
  owner: "",
  repo: "",
  branch: "main",
  rootDir: "LeetCode",
  autoSync: true,
  updateRootReadme: true,
  interviewMode: false,
  promptNotes: true
};

export const DEFAULTS = {
  settings: DEFAULT_SETTINGS,
  queue: [],
  problems: {},
  lastSync: null,
  lastError: "",
  synced: {},
  syncedProblems: {},
  reviews: {}
};

function extensionApi() {
  if (globalThis.browser?.storage?.local) return globalThis.browser;
  return globalThis.chrome;
}

function normalizeError(error) {
  if (error instanceof Error) return error;
  if (error?.message) return new Error(error.message);
  return new Error(String(error));
}

function promisifyStorage(method, argument) {
  const api = extensionApi();
  const storage = api?.storage?.local;
  if (!storage || typeof storage[method] !== "function") {
    return Promise.reject(new Error("Extension storage is unavailable."));
  }

  const promiseStyle = Boolean(globalThis.browser?.storage?.local);

  return new Promise((resolve, reject) => {
    let settled = false;
    const finish = (value, error) => {
      if (settled) return;
      settled = true;
      if (error) reject(normalizeError(error));
      else resolve(value);
    };

    let returned;
    try {
      if (promiseStyle) {
        returned = storage[method](argument);
      } else {
        returned = storage[method](argument, (value) => {
          const failure = api.runtime?.lastError;
          finish(value, failure ? normalizeError(failure) : null);
        });
      }
    } catch (error) {
      finish(undefined, error);
      return;
    }

    if (returned && typeof returned.then === "function") {
      returned.then((value) => finish(value), (error) => finish(undefined, error));
    } else if (promiseStyle) {
      finish(returned, null);
    }
  });
}

function hasEntries(value) {
  return Boolean(value) && typeof value === "object" && Object.keys(value).length > 0;
}

function readLegacyKeys(raw) {
  return hasEntries(raw?.synced) || hasEntries(raw?.syncedProblems) || hasEntries(raw?.reviews);
}

export function migrateLegacyState(raw) {
  if (!readLegacyKeys(raw)) return null;
  const problems = { ...(raw.problems || {}) };

  for (const [key, value] of Object.entries(raw.syncedProblems || {})) {
    if (!value || typeof value !== "object") continue;
    problems[key] = {
      ...value,
      ...(problems[key] || {}),
      syncedAt: value.syncedAt || raw.synced?.[key] || value.acceptedAt || ""
    };
  }

  for (const [key, acceptedAt] of Object.entries(raw.synced || {})) {
    if (problems[key]) {
      problems[key].syncedAt = problems[key].syncedAt || acceptedAt || "";
      continue;
    }
    const [number = "", language = ""] = String(key).split(":");
    problems[key] = { number, language, syncedAt: acceptedAt || "" };
  }

  for (const [key, value] of Object.entries(raw.reviews || {})) {
    if (!value || typeof value !== "object") continue;
    const existing = problems[key] || {};
    problems[key] = { ...existing, ...value, review: value.review || existing.review };
  }

  return problems;
}

function normalizeSolution(solution) {
  const source = solution && typeof solution === "object" ? solution : {};
  return {
    approach: typeof source.approach === "string" ? source.approach.trim() : "",
    code: typeof source.code === "string" ? source.code : "",
    notes: typeof source.notes === "string" ? source.notes : "",
    timeComplexity: typeof source.timeComplexity === "string" ? source.timeComplexity : "",
    spaceComplexity: typeof source.spaceComplexity === "string" ? source.spaceComplexity : "",
    acceptedAt: typeof source.acceptedAt === "string" ? source.acceptedAt : ""
  };
}

export function normalizeProblem(problem) {
  if (!problem || typeof problem !== "object") return problem;
  if (Array.isArray(problem.solutions) && problem.solutions.length) {
    return { ...problem, solutions: problem.solutions.map(normalizeSolution) };
  }
  const entry = normalizeSolution(problem);
  return {
    ...problem,
    solutions: [entry],
    code: entry.code,
    notes: entry.notes,
    timeComplexity: entry.timeComplexity,
    spaceComplexity: entry.spaceComplexity
  };
}

export async function loadState() {
  const raw = (await promisifyStorage("get", DEFAULTS)) || {};
  const legacyProblems = migrateLegacyState(raw);
  const source = legacyProblems || raw.problems || {};
  const problems = Object.fromEntries(
    Object.entries(source).map(([key, value]) => [key, normalizeProblem(value)])
  );

  return {
    settings: { ...DEFAULT_SETTINGS, ...(raw.settings || {}) },
    queue: Array.isArray(raw.queue) ? raw.queue : [],
    problems,
    lastSync: raw.lastSync || null,
    lastError: typeof raw.lastError === "string" ? raw.lastError : "",
    migrated: Boolean(legacyProblems)
  };
}

export async function saveState(state) {
  const safe = state || {};
  const payload = {
    settings: { ...DEFAULT_SETTINGS, ...(safe.settings || {}) },
    queue: Array.isArray(safe.queue) ? safe.queue : [],
    problems: safe.problems && typeof safe.problems === "object" ? safe.problems : {},
    lastSync: safe.lastSync || null,
    lastError: typeof safe.lastError === "string" ? safe.lastError : "",
    synced: {},
    syncedProblems: {},
    reviews: {}
  };
  await promisifyStorage("set", payload);
  return payload;
}
