import { dateOnly } from "./review.js";

function problemList(source) {
  return Array.isArray(source) ? source.filter(Boolean) : Object.values(source || {}).filter(Boolean);
}

function dayKey(value) {
  if (!value) return "";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "" : dateOnly(date);
}

function shiftDay(day, amount) {
  const date = new Date(`${day}T00:00:00Z`);
  date.setUTCDate(date.getUTCDate() + amount);
  return dateOnly(date);
}

export function collectActivity(problems) {
  const days = new Map();

  const bump = (day, field) => {
    if (!day) return;
    const current = days.get(day) || { solves: 0, reviews: 0 };
    current[field] += 1;
    days.set(day, current);
  };

  for (const problem of problemList(problems)) {
    bump(dayKey(problem.acceptedAt), "solves");
    const history = Array.isArray(problem.review?.history) ? problem.review.history : [];
    for (const entry of history) bump(dayKey(entry?.at), "reviews");
  }

  return days;
}

export function streakStats(problems, today = dateOnly()) {
  const days = collectActivity(problems);
  const active = [...days.keys()].sort();
  const hasActivity = (day) => days.has(day);

  let longest = 0;
  let run = 0;
  let previous = "";
  for (const day of active) {
    run = previous && shiftDay(previous, 1) === day ? run + 1 : 1;
    longest = Math.max(longest, run);
    previous = day;
  }

  let cursor = hasActivity(today) ? today : shiftDay(today, -1);
  let current = 0;
  while (hasActivity(cursor)) {
    current += 1;
    cursor = shiftDay(cursor, -1);
  }

  return { current, longest, activeDays: active.length, todayActive: hasActivity(today) };
}

export function heatmapWeeks(problems, options = {}) {
  const weeks = Math.max(1, options.weeks || 52);
  const today = options.today || dateOnly();
  const days = collectActivity(problems);

  const weekday = new Date(`${today}T00:00:00Z`).getUTCDay();
  const end = shiftDay(today, 6 - weekday);
  let cursor = shiftDay(end, -(weeks * 7 - 1));

  const result = [];
  for (let week = 0; week < weeks; week++) {
    const cells = [];
    for (let day = 0; day < 7; day++) {
      const counts = days.get(cursor) || { solves: 0, reviews: 0 };
      cells.push({
        date: cursor,
        solves: counts.solves,
        reviews: counts.reviews,
        total: counts.solves + counts.reviews,
        future: cursor > today
      });
      cursor = shiftDay(cursor, 1);
    }
    result.push(cells);
  }
  return result;
}

const GRADED = ["again", "hard", "good", "easy"];

export function gradeStats(problems) {
  const counts = { again: 0, hard: 0, good: 0, easy: 0, skip: 0 };

  for (const problem of problemList(problems)) {
    const review = problem.review || {};
    const history = Array.isArray(review.history) ? review.history : [];
    if (history.length) {
      for (const entry of history) {
        if (entry?.grade in counts) counts[entry.grade] += 1;
      }
    } else if (review.lastGrade in counts) {
      // Records graded before history existed only carry the last grade.
      counts[review.lastGrade] += 1;
    }
  }

  const total = GRADED.reduce((sum, key) => sum + counts[key], 0);
  const recalled = counts.hard + counts.good + counts.easy;
  return { counts, total, recallRate: total ? recalled / total : 0 };
}

export function difficultyStats(problems) {
  const counts = { Easy: 0, Medium: 0, Hard: 0, Unknown: 0 };
  for (const problem of problemList(problems)) {
    const key = ["Easy", "Medium", "Hard"].includes(problem.difficulty)
      ? problem.difficulty
      : "Unknown";
    counts[key] += 1;
  }
  return counts;
}

export function tagStats(problems, limit = 12) {
  const counts = new Map();
  for (const problem of problemList(problems)) {
    for (const tag of problem.tags || []) {
      if (!tag) continue;
      counts.set(tag, (counts.get(tag) || 0) + 1);
    }
  }
  return [...counts.entries()]
    .map(([tag, count]) => ({ tag, count }))
    .sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag))
    .slice(0, limit);
}
