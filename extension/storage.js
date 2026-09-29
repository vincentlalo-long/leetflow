export const DEFAULTS = {
  settings: {
    githubToken: "",
    owner: "",
    repo: "",
    branch: "main",
    rootDir: "LeetCode",
    autoSync: true,
    updateRootReadme: true,
    interviewMode: false,
    promptNotes: true
  },
  queue: [],
  synced: {},
  syncedProblems: {},
  reviews: {}
};

function storageGet(keys) {
  if (typeof browser !== "undefined" && browser?.storage?.local?.get) {
    return browser.storage.local.get(keys);
  }
  return new Promise((resolve, reject) => {
    try {
      chrome.storage.local.get(keys, (result) => {
        if (chrome.runtime?.lastError) {
          reject(chrome.runtime.lastError);
        } else {
          resolve(result || {});
        }
      });
    } catch (err) {
      reject(err);
    }
  });
}

function storageSet(items) {
  if (typeof browser !== "undefined" && browser?.storage?.local?.set) {
    return browser.storage.local.set(items);
  }
  return new Promise((resolve, reject) => {
    try {
      chrome.storage.local.set(items, () => {
        if (chrome.runtime?.lastError) {
          reject(chrome.runtime.lastError);
        } else {
          resolve();
        }
      });
    } catch (err) {
      reject(err);
    }
  });
}

export async function loadState() {
  const rawState = await storageGet(DEFAULTS);
  const state = rawState || {};
  return {
    settings: { ...DEFAULTS.settings, ...(state.settings || {}) },
    queue: Array.isArray(state.queue) ? state.queue : [],
    synced: state.synced || {},
    syncedProblems: state.syncedProblems || {},
    reviews: state.reviews || {}
  };
}

export async function saveState(state) {
  const safeState = state || {};
  await storageSet({
    settings: safeState.settings || DEFAULTS.settings,
    queue: Array.isArray(safeState.queue) ? safeState.queue : [],
    synced: safeState.synced || {},
    syncedProblems: safeState.syncedProblems || {},
    reviews: safeState.reviews || {}
  });
}
