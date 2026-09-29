(function () {
  function getMonacoData() {
    try {
      if (window.monaco?.editor) {
        const models = window.monaco.editor.getModels?.() || [];
        if (models.length > 0) {
          let best = models[0];
          for (const m of models) {
            const val = m.getValue?.() || "";
            if (val.length > (best.getValue?.()?.length || 0)) {
              best = m;
            }
          }
          const code = best.getValue?.() || "";
          const lang = best.getLanguageId?.() || best.getModeId?.() || "";
          return { code, lang };
        }
      }
    } catch (e) {
      console.error("[LeetFlow] Monaco extraction error:", e);
    }
    return null;
  }

  function syncToDOM() {
    const data = getMonacoData();
    if (data && data.code) {
      document.documentElement.dataset.leetflowCode = data.code;
      if (data.lang) {
        document.documentElement.dataset.leetflowLang = data.lang;
      }
    }
  }

  window.addEventListener("leetflow:request-editor", () => {
    syncToDOM();
    const data = getMonacoData();
    window.dispatchEvent(new CustomEvent("leetflow:response-editor", { detail: data }));
  });

  // Sync periodically while coding
  setInterval(syncToDOM, 1000);
  syncToDOM();
})();
