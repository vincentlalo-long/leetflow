export function fakeResponse(status, body, headers = {}) {
  const normalized = Object.fromEntries(
    Object.entries(headers).map(([key, value]) => [key.toLowerCase(), value])
  );
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: status >= 200 && status < 300 ? "OK" : "Error",
    headers: { get: (name) => normalized[String(name).toLowerCase()] ?? null },
    text: async () => (body === undefined || body === null ? "" : JSON.stringify(body)),
    json: async () => body
  };
}

export function createFetchRouter() {
  const calls = [];
  const routes = [];

  const router = {
    calls,
    route(method, pattern, handler) {
      routes.push({ method: String(method).toUpperCase(), pattern, handler });
      return router;
    },
    reset() {
      calls.length = 0;
    },
    install() {
      globalThis.fetch = async (url, options = {}) => {
        const method = String(options.method || "GET").toUpperCase();
        const pathname = new URL(url).pathname;
        let body = null;
        if (options.body) {
          try {
            body = JSON.parse(options.body);
          } catch {
            body = options.body;
          }
        }
        const call = { method, pathname, body, url: String(url) };
        calls.push(call);

        for (const route of routes) {
          if (route.method !== method) continue;
          const matched =
            typeof route.pattern === "string"
              ? pathname === route.pattern
              : route.pattern.test(pathname);
          if (!matched) continue;
          const result =
            typeof route.handler === "function" ? route.handler(call, calls) : route.handler;
          return fakeResponse(result?.status ?? 200, result?.body ?? {}, result?.headers ?? {});
        }

        return fakeResponse(404, { message: `Unhandled ${method} ${pathname}` });
      };
    },
    restore() {
      delete globalThis.fetch;
    }
  };

  router.install();
  return router;
}

export function installChromeMock(initialStore = {}) {
  const store = { ...initialStore };
  const listeners = {};
  const alarms = [];

  const chrome = {
    runtime: {
      lastError: null,
      onMessage: { addListener: (fn) => (listeners.message = fn) },
      onStartup: { addListener: (fn) => (listeners.startup = fn) },
      onInstalled: { addListener: (fn) => (listeners.installed = fn) },
      openOptionsPage: () => {},
      getURL: (path) => `chrome-extension://test/${path}`
    },
    storage: {
      local: {
        get(keys, callback) {
          let result = {};
          if (Array.isArray(keys)) {
            for (const key of keys) if (key in store) result[key] = store[key];
          } else if (keys && typeof keys === "object") {
            for (const [key, fallback] of Object.entries(keys)) {
              result[key] = key in store ? store[key] : fallback;
            }
          } else {
            result = { ...store };
          }
          if (callback) queueMicrotask(() => callback(result));
          return Promise.resolve(result);
        },
        set(items, callback) {
          Object.assign(store, items);
          if (callback) queueMicrotask(() => callback());
          return Promise.resolve();
        }
      }
    },
    alarms: {
      create: (name, options) => alarms.push({ name, options }),
      onAlarm: { addListener: (fn) => (listeners.alarm = fn) }
    }
  };

  globalThis.chrome = chrome;
  return { store, listeners, alarms, chrome };
}

export function configureRepository(settings = {}) {
  return {
    settings: {
      githubToken: "test-token",
      owner: "acme",
      repo: "leetcode",
      branch: "main",
      rootDir: "LeetCode",
      autoSync: true,
      updateRootReadme: true,
      interviewMode: false,
      promptNotes: false,
      ...settings
    },
    queue: [],
    problems: {},
    lastSync: null,
    lastError: "",
    synced: {},
    syncedProblems: {},
    reviews: {}
  };
}

export function registerGithubRoutes(router, repo, options = {}) {
  let blobs = 0;
  let patchFailures = options.patchFailures || 0;
  const nextSha = () => `sha${++blobs}`;

  const trees = new Map();
  const commits = new Map();
  const state = { head: null };

  if (options.initialHead) {
    const treeSha = nextSha();
    trees.set(treeSha, new Map());
    commits.set(options.initialHead, { tree: treeSha, parents: [] });
    state.head = options.initialHead;
    repo.head = options.initialHead;
  }

  function baseEntries(treeSha) {
    return treeSha && trees.has(treeSha) ? trees.get(treeSha) : new Map();
  }

  function currentTreeSha() {
    if (!state.head) return null;
    const commit = commits.get(state.head);
    return commit ? commit.tree : null;
  }

  function materialized() {
    const entries = baseEntries(currentTreeSha());
    const output = new Map();
    for (const [path, entry] of entries) output.set(path, entry);
    return output;
  }

  router.route("GET", /\/git\/ref\/heads\/[^/]+$/, () => {
    if (!state.head) return { status: 404, body: { message: "Not Found" } };
    return { status: 200, body: { ref: "refs/heads/main", object: { sha: state.head } } };
  });

  router.route("GET", /\/git\/commits\/[^/]+$/, (call) => {
    const sha = call.pathname.split("/").pop();
    const commit = commits.get(sha);
    if (!commit) return { status: 404, body: { message: "Not Found" } };
    return { status: 200, body: { sha, tree: { sha: commit.tree } } };
  });

  router.route("POST", /\/git\/trees$/, (call) => {
    const merged = new Map(baseEntries(call.body?.base_tree));
    for (const entry of call.body?.tree || []) {
      merged.set(entry.path, { mode: entry.mode, type: entry.type, content: entry.content ?? null });
    }
    const sha = nextSha();
    trees.set(sha, merged);
    return { status: 201, body: { sha } };
  });

  router.route("POST", /\/git\/commits$/, (call) => {
    const sha = nextSha();
    commits.set(sha, { tree: call.body.tree, parents: call.body.parents || [] });
    repo.commits.push({ sha, message: call.body.message, tree: call.body.tree });
    return { status: 201, body: { sha, tree: { sha: call.body.tree } } };
  });

  router.route("PATCH", /\/git\/refs\/heads\/[^/]+$/, (call) => {
    if (patchFailures > 0) {
      patchFailures--;
      return { status: 422, body: { message: "Update is not a fast forward" } };
    }
    const target = call.body?.sha;
    if (!commits.has(target)) return { status: 422, body: { message: "Object does not exist" } };
    state.head = target;
    repo.head = target;
    return { status: 200, body: { ref: "refs/heads/main", object: { sha: target } } };
  });

  router.route("POST", /\/git\/refs$/, (call) => {
    const target = call.body?.sha;
    if (state.head) return { status: 422, body: { message: "Reference already exists" } };
    if (!commits.has(target)) return { status: 422, body: { message: "Object does not exist" } };
    state.head = target;
    repo.head = target;
    return { status: 201, body: { ref: call.body.ref, object: { sha: target } } };
  });

  router.route("GET", /\/contents\/.+$/, (call) => {
    const path = decodeURIComponent(call.pathname.split("/contents/")[1]);
    const entry = materialized().get(path);
    if (!entry || entry.content === null) {
      return { status: 404, body: { message: "Not Found" } };
    }
    return {
      status: 200,
      body: {
        path,
        encoding: "base64",
        content: Buffer.from(String(entry.content), "utf8").toString("base64")
      }
    };
  });

  return {
    state,
    read(path) {
      const entry = materialized().get(path);
      return entry && entry.content !== null ? String(entry.content) : "";
    },
    files() {
      return [...materialized().keys()].sort();
    },
    commits: repo.commits
  };
}

export function registerGraphqlRoute(router, answers) {
  router.route("POST", /\/graphql$/, (call) => {
    const slug = call.body?.variables?.titleSlug;
    const answer = typeof answers === "function" ? answers(slug) : answers;
    return { status: 200, body: { data: { question: answer } } };
  });
}
