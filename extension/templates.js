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
  const num = /^\d+$/.test(String(problem.number))
    ? String(problem.number).padStart(4, "0")
    : String(problem.number || "0000");
  const slug = slugify(problem.slug || problem.title);
  return `${num}-${slug}`;
}

export function problemPath(problem, settings) {
  const folder = problemFolder(problem);
  const root = settings?.rootDir ? `${settings.rootDir.replace(/\/+$/, "")}/` : "";
  return `${root}${folder}`;
}

export function renderProblemReadme(problem, settings) {
  const tags = (problem.tags || []).map((tag) => `[${tag}](../README.md#tag-${slugify(tag)})`).join(", ") || "-";
  const ext = problem.extension || normalizeLanguage(problem.language).ext || "txt";
  const solution = `${slugify(problem.slug || problem.title)}.${ext}`;
  const cleanTitle = String(problem.title || "").replace(/^\d+\.\s*/, "").trim();
  const roadmaps = getRoadmapBadges(problem.number);
  const badgeLine = [badge(problem.difficulty), roadmaps].filter(Boolean).join(" ");

  const notesSection = problem.notes
    ? `> 💡 **Aha! Moment / Key Insight**:\n> ${problem.notes.replace(/\n/g, "\n> ")}`
    : `_Add your notes here._`;

  return `# [${problem.number}. ${cleanTitle}](${problem.url})

> ${badgeLine}

| Property | Value |
|---|---|
| Difficulty | ${escapeCell(problem.difficulty)} |
| Topics | ${tags} |
| Language | ${escapeCell(problem.language)} |
| Status | Accepted |
| Time Spent | ${escapeCell(problem.timeSpent || "N/A")} |
| Attempts | ${escapeCell(problem.attemptsSummary || "1 (Clean AC)")} |
| Synced | ${escapeCell(problem.acceptedAt)} |

---

## Problem

${problem.description || "_Problem statement was not available in the page payload._"}

---

## Solution

[\`${solution}\`](./${solution})

## Notes

${notesSection}
`;
}

export function renderRootReadme(problems, settings) {
  const groups = new Map();
  for (const problem of problems) {
    for (const tag of problem.tags || []) {
      const key = tag.toLowerCase();
      if (!groups.has(key)) groups.set(key, { name: tag, problems: [] });
      groups.get(key).problems.push(problem);
    }
  }
  const tags = [...groups.values()].sort((a, b) => a.name.localeCompare(b.name));
  const rows = [...problems].sort((a, b) => Number(a.number) - Number(b.number))
    .map((p) => {
      const folder = problemFolder(p);
      const cleanTitle = String(p.title || "").replace(/^\d+\.\s*/, "").trim();
      return `| ${p.number} | [${cleanTitle}](./${folder}/README.md) | ${badge(p.difficulty)} | ${escapeCell(p.language)} |`;
    })
    .join("\n");

  const tagSections = tags.map((group) => {
    const items = group.problems.sort((a, b) => Number(a.number) - Number(b.number))
      .map((p) => {
        const folder = problemFolder(p);
        const cleanTitle = String(p.title || "").replace(/^\d+\.\s*/, "").trim();
        return `| [${p.number}. ${cleanTitle}](./${folder}/README.md) | ${badge(p.difficulty)} |`;
      })
      .join("\n");
    return `<a id="tag-${slugify(group.name)}"></a>

### ${group.name} (${group.problems.length})

| Problem | Difficulty |
|---|---|
${items}`;
  }).join("\n\n");

  return `# LeetCode Solutions

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
