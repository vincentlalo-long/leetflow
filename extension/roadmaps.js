import { LEETCODE_75, TOP_100_LIKED, TOP_INTERVIEW_150 } from "./plans.js";

export const BLIND_75 = new Set([
  1, 3, 5, 11, 15, 19, 20, 21, 23, 33, 39, 48, 49, 53, 54, 55, 56, 57, 62, 70,
  73, 76, 79, 91, 98, 100, 102, 104, 105, 121, 124, 125, 128, 133, 139, 141,
  143, 152, 153, 190, 191, 198, 200, 206, 207, 208, 211, 212, 213, 217, 226,
  230, 235, 238, 242, 252, 253, 261, 268, 269, 271, 295, 297, 300, 322, 323, 338,
  347, 371, 417, 424, 435, 572, 647, 1143
]);

export const NEETCODE_150 = new Set([
  ...BLIND_75,
  2, 4, 7, 10, 17, 22, 25, 36, 40, 42, 43, 45, 46, 50, 51, 66, 72, 74, 78, 84, 90,
  97, 110, 115, 127, 130, 131, 134, 136, 138, 146, 150, 155, 167, 199, 202, 210,
  215, 239, 286, 287, 309, 312, 329, 332, 355, 416, 494, 518, 543, 567, 621,
  678, 684, 695, 703, 704, 739, 743, 746, 763, 778, 787, 846, 853, 875, 973,
  981, 994, 1046, 1448, 1584, 1851, 1899, 2013
]);

export const ROADMAPS = [
  { id: "leetcode-75", name: "LeetCode 75", color: "#f89f1b", problems: LEETCODE_75 },
  { id: "top-interview-150", name: "Top Interview 150", color: "#ef4743", problems: TOP_INTERVIEW_150 },
  { id: "top-100-liked", name: "Top 100 Liked", color: "#1a7f37", problems: TOP_100_LIKED },
  { id: "blind-75", name: "Blind 75", color: "#007acc", problems: BLIND_75 },
  { id: "neetcode-150", name: "NeetCode 150", color: "#6366f1", problems: NEETCODE_150 }
];

function roadmapsFor(number) {
  const value = Number(number);
  if (Number.isNaN(value)) return [];
  return ROADMAPS.filter((plan) => plan.problems.has(value));
}

export function getRoadmaps(number) {
  return roadmapsFor(number).map((plan) => plan.name);
}

export function getRoadmapBadges(number) {
  return roadmapsFor(number)
    .map((plan) => {
      const label = encodeURIComponent(plan.name.replace(/-/g, "--"));
      const color = plan.color.replace("#", "");
      return `![${plan.name}](https://img.shields.io/badge/Roadmap-${label}-${color}?style=flat-square)`;
    })
    .join(" ");
}

export function getRoadmapProgress(syncedProblems = []) {
  const solvedSet = new Set(
    syncedProblems
      .map((problem) => Number(problem?.number))
      .filter((number) => !Number.isNaN(number) && number > 0)
  );

  return ROADMAPS.map((plan) => {
    let solved = 0;
    for (const number of plan.problems) {
      if (solvedSet.has(number)) solved += 1;
    }
    const total = plan.problems.size;
    return {
      id: plan.id,
      name: plan.name,
      color: plan.color,
      solved,
      total,
      percent: total ? Math.round((solved / total) * 100) : 0
    };
  });
}
