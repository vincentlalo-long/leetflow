import { getRoadmapBadges } from "./roadmaps.js";

export function slugify(value) {
  return String(value || "")
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export function normalizeLanguage(raw) {
  if (!raw) return { name: "Unknown", ext: "txt" };
  const clean = String(raw).trim().toLowerCase();

  const map = {
    "c++": { name: "C++", ext: "cpp" },
    cpp: { name: "C++", ext: "cpp" },
    c: { name: "C", ext: "c" },
    java: { name: "Java", ext: "java" },
    python: { name: "Python", ext: "py" },
    python3: { name: "Python3", ext: "py" },
    py: { name: "Python3", ext: "py" },
    csharp: { name: "C#", ext: "cs" },
    "c#": { name: "C#", ext: "cs" },
    cs: { name: "C#", ext: "cs" },
    javascript: { name: "JavaScript", ext: "js" },
    js: { name: "JavaScript", ext: "js" },
    typescript: { name: "TypeScript", ext: "ts" },
    ts: { name: "TypeScript", ext: "ts" },
    golang: { name: "Go", ext: "go" },
    go: { name: "Go", ext: "go" },
    rust: { name: "Rust", ext: "rs" },
    rs: { name: "Rust", ext: "rs" },
    swift: { name: "Swift", ext: "swift" },
    kotlin: { name: "Kotlin", ext: "kt" },
    kt: { name: "Kotlin", ext: "kt" },
    ruby: { name: "Ruby", ext: "rb" },
    rb: { name: "Ruby", ext: "rb" },
    scala: { name: "Scala", ext: "scala" },
    php: { name: "PHP", ext: "php" },
    dart: { name: "Dart", ext: "dart" },
    elixir: { name: "Elixir", ext: "ex" },
    erlang: { name: "Erlang", ext: "erl" },
    racket: { name: "Racket", ext: "rkt" },
    mysql: { name: "MySQL", ext: "sql" },
    sql: { name: "SQL", ext: "sql" },
    "ms sql server": { name: "MS SQL Server", ext: "sql" },
    oracle: { name: "Oracle", ext: "sql" },
    postgresql: { name: "PostgreSQL", ext: "sql" },
    bash: { name: "Bash", ext: "sh" }
  };

  if (map[clean]) return map[clean];

  for (const [key, val] of Object.entries(map)) {
    if (clean.includes(key)) return val;
  }

  return { name: raw, ext: "txt" };
}

function badge(difficulty) {
  const value = String(difficulty || "Unknown");
  const color = { Easy: "2ea44f", Medium: "f0ad4e", Hard: "d73a49" }[value] || "6e7781";
  return `![${value}](https://img.shields.io/badge/difficulty-${value}-${color}?style=flat-square)`;
}

function escapeCell(value) {
  return String(value || "-").replace(/\|/g, "\\|").replace(/\n/g, " ");
}

export function problemFolder(problem) {
  const raw = String(problem.number || "").trim();
  const num = /^\d+$/.test(raw)
    ? raw.padStart(4, "0")
    : "0000";
  const slug = slugify(problem.slug || problem.title);
  return num !== "0000" ? `${num}-${slug}` : slug;
}

export function problemPath(problem, settings) {
  const folder = problemFolder(problem);
  const root = settings?.rootDir ? `${settings.rootDir.replace(/\/+$/, "")}/` : "";
  return `${root}${folder}`;
}

export function solutionFiles(problem) {
  const base = slugify(problem.slug || problem.title);
  const solutions =
    Array.isArray(problem.solutions) && problem.solutions.length
      ? problem.solutions
      : [{ approach: "", code: problem.code ?? "" }];

  const used = new Set();
  return solutions.map((solution) => {
    const extension =
      solution.extension || problem.extension || normalizeLanguage(problem.language).ext || "txt";
    const label = slugify(solution.approach);
    const stem = label ? `${base}-${label}` : base;
    let file = `${stem}.${extension}`;
    let counter = 2;
    while (used.has(file)) {
      file = `${stem}-v${counter}.${extension}`;
      counter += 1;
    }
    used.add(file);
    return { solution, file };
  });
}

export function mergeFolderRecords(records) {
  const list = (records || []).filter((record) => record && (record.title || record.slug));
  if (list.length <= 1) return list[0] || null;

  const primary = [...list].sort((a, b) =>
    String(b.syncedAt || b.acceptedAt || "").localeCompare(String(a.syncedAt || a.acceptedAt || ""))
  )[0];
  const languages = [...new Set(list.map((record) => record.language).filter(Boolean))];
  const tags = [...new Set(list.flatMap((record) => record.tags || []))];
  const solutions = list.flatMap((record) => {
    const own =
      Array.isArray(record.solutions) && record.solutions.length
        ? record.solutions
        : [
            {
              approach: "",
              code: record.code || "",
              notes: record.notes || "",
              timeComplexity: record.timeComplexity || "",
              spaceComplexity: record.spaceComplexity || ""
            }
          ];
    return own.map((solution) => ({
      ...solution,
      language: solution.language || record.language,
      extension: solution.extension || record.extension || normalizeLanguage(record.language).ext
    }));
  });

  return { ...primary, language: languages.join(", "), tags, solutions };
}

export function renderProblemReadme(problem, settings) {
  const tags = (problem.tags || []).map((tag) => `[${tag}](../README.md#tag-${slugify(tag)})`).join(", ") || "-";
  const entries = solutionFiles(problem);
  const single = entries.length === 1 && !String(entries[0].solution.approach || "").trim();
  const cleanTitle = String(problem.title || "").replace(/^\d+\.\s*/, "").trim();
  const roadmaps = getRoadmapBadges(problem.number);
  const badgeLine = [badge(problem.difficulty), roadmaps].filter(Boolean).join(" ");

  const notesSection = problem.notes
    ? `> 💡 **Aha! Moment / Key Insight**:\n> ${problem.notes.replace(/\n/g, "\n> ")}`
    : `_Add your notes here._`;

  const numPrefix = /^\d+$/.test(String(problem.number)) ? `${problem.number}. ` : "";

  const complexityRows = single
    ? [
        problem.timeComplexity ? `| Time Complexity | \`${escapeCell(problem.timeComplexity)}\` |` : "",
        problem.spaceComplexity ? `| Space Complexity | \`${escapeCell(problem.spaceComplexity)}\` |` : ""
      ]
        .filter(Boolean)
        .join("\n")
    : "";

  const solutionsSection = single
    ? `## Solution

[\`${entries[0].file}\`](./${entries[0].file})

## Notes

${notesSection}`
    : `## Solutions

${entries.map(({ solution, file }) => {
  const heading =
    String(solution.approach || "").replace(/\n/g, " ").trim() ||
    String(solution.language || "").replace(/\n/g, " ").trim() ||
    "Default";
  const rows = [
    solution.timeComplexity ? `- **Time**: \`${escapeCell(solution.timeComplexity)}\`` : "",
    solution.spaceComplexity ? `- **Space**: \`${escapeCell(solution.spaceComplexity)}\`` : ""
  ]
    .filter(Boolean)
    .join("\n");
  const notes = solution.notes
    ? `\n\n> 💡 ${solution.notes.replace(/\n/g, "\n> ")}`
    : "";
  return `### ${heading}

[\`${file}\`](./${file})${rows ? `\n\n${rows}` : ""}${notes}`;
}).join("\n\n")}`;

  return `# [${numPrefix}${cleanTitle}](${problem.url})

> ${badgeLine}

| Property | Value |
|---|---|
| Difficulty | ${escapeCell(problem.difficulty)} |
| Topics | ${tags} |
| Language | ${escapeCell(problem.language)} |
${complexityRows ? `${complexityRows}\n` : ""}| Status | Accepted |
| Time Spent | ${escapeCell(problem.timeSpent || "N/A")} |
| Attempts | ${escapeCell(problem.attemptsSummary || "1 (Clean AC)")} |
| Synced | ${escapeCell(problem.acceptedAt)} |

---

## Problem

${problem.description || "_Problem statement was not available in the page payload._"}

---

${solutionsSection}
`;
}

export function renderRootReadme(problems, settings) {
  const usable = (problems || []).filter(
    (problem) => problem && (problem.title || problem.slug) && problem.syncedAt
  );
  const folders = new Map();
  for (const problem of usable) {
    const folder = problemFolder(problem);
    if (!folders.has(folder)) folders.set(folder, []);
    folders.get(folder).push(problem);
  }
  const tracked = [...folders.values()]
    .map((records) => mergeFolderRecords(records))
    .filter(Boolean);
  const groups = new Map();
  for (const problem of tracked) {
    for (const tag of problem.tags || []) {
      const key = tag.toLowerCase();
      if (!groups.has(key)) groups.set(key, { name: tag, problems: [] });
      groups.get(key).problems.push(problem);
    }
  }
  const tags = [...groups.values()].sort((a, b) => a.name.localeCompare(b.name));
  const rows = [...tracked].sort((a, b) => Number(a.number || 0) - Number(b.number || 0))
    .map((p) => {
      const folder = problemFolder(p);
      const cleanTitle = String(p.title || "").replace(/^\d+\.\s*/, "").trim();
      const numDisplay = /^\d+$/.test(String(p.number)) ? p.number : "-";
      return `| ${numDisplay} | [${cleanTitle}](./${folder}/README.md) | ${badge(p.difficulty)} | ${escapeCell(p.language)} |`;
    })
    .join("\n");

  const tagSections = tags.map((group) => {
    const items = group.problems.sort((a, b) => Number(a.number || 0) - Number(b.number || 0))
      .map((p) => {
        const folder = problemFolder(p);
        const cleanTitle = String(p.title || "").replace(/^\d+\.\s*/, "").trim();
        const numPrefix = /^\d+$/.test(String(p.number)) ? `${p.number}. ` : "";
        return `| [${numPrefix}${cleanTitle}](./${folder}/README.md) | ${badge(p.difficulty)} |`;
      })
      .join("\n");
    return `<a id="tag-${slugify(group.name)}"></a>

### ${group.name} (${group.problems.length})

| Problem | Difficulty |
|---|---|
${items}`;
  }).join("\n\n");

  return `<!-- LEETFLOW:START -->
# LeetCode Solutions

> Generated by LeetFlow Sync. Solve on LeetCode, keep the archive on GitHub.

## Progress

| # | Problem | Difficulty | Language |
|---:|---|---|---|
${rows || "| - | No synced problems yet. | - | - |"}

## Browse by Tag

${tagSections || "_No topic tags yet._"}
<!-- LEETFLOW:END -->
`;
}

export const ROOT_START = "<!-- LEETFLOW:START -->";
export const ROOT_END = "<!-- LEETFLOW:END -->";

function isLegacyGeneratedReadme(content) {
  return content.includes("Generated by LeetFlow Sync") && !content.includes(ROOT_START);
}

export function mergeRootReadme(existing, generated) {
  const block = String(generated || "").trim();
  const current = String(existing || "");

  if (!current.trim()) return `${block}\n`;
  if (isLegacyGeneratedReadme(current)) return `${block}\n`;

  const start = current.indexOf(ROOT_START);
  const end = current.indexOf(ROOT_END);
  if (start !== -1 && end > start) {
    return (
      current.slice(0, start) +
      block +
      current.slice(end + ROOT_END.length)
    );
  }

  return `${current.replace(/\s+$/, "")}\n\n${block}\n`;
}
