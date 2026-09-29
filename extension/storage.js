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

export async function loadState() {
  const state = await chrome.storage.local.get(DEFAULTS);
  return {
    settings: { ...DEFAULTS.settings, ...(state.settings || {}) },
    queue: Array.isArray(state.queue) ? state.queue : [],
    synced: state.synced || {},
    syncedProblems: state.syncedProblems || {},
    reviews: state.reviews || {}
  };
}

export async function saveState(state) {
  await chrome.storage.local.set({
    settings: state.settings,
    queue: state.queue,
    synced: state.synced,
    syncedProblems: state.syncedProblems || {},
    reviews: state.reviews || {}
  });
}
