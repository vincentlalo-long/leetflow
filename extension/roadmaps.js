export const BLIND_75 = new Set([
  1, 3, 5, 11, 15, 19, 20, 21, 23, 33, 39, 48, 49, 53, 54, 55, 56, 57, 62, 70,
  73, 76, 79, 91, 98, 100, 102, 104, 105, 121, 124, 125, 128, 133, 139, 141,
  143, 152, 153, 190, 191, 198, 200, 206, 207, 208, 211, 212, 213, 217, 226,
  230, 235, 238, 242, 252, 253, 261, 268, 269, 295, 297, 300, 322, 323, 338,
  347, 371, 417, 424, 435, 572, 647, 1143
]);

export const NEETCODE_150 = new Set([
  ...BLIND_75,
  2, 4, 7, 10, 17, 22, 36, 40, 42, 43, 45, 46, 50, 51, 66, 72, 74, 78, 84, 90,
  97, 110, 115, 130, 131, 134, 136, 138, 146, 150, 155, 167, 199, 202, 210,
  215, 239, 286, 287, 309, 312, 329, 332, 355, 416, 494, 518, 543, 567, 621,
  678, 684, 695, 703, 704, 739, 743, 746, 763, 778, 787, 846, 853, 875, 973,
  981, 994, 1046, 1448, 1584, 1851, 1899, 2013
]);

export function getRoadmaps(number) {
  const num = Number(number);
  const roadmaps = [];
  if (BLIND_75.has(num)) roadmaps.push("Blind 75");
  if (NEETCODE_150.has(num)) roadmaps.push("NeetCode 150");
  return roadmaps;
}

export function getRoadmapBadges(number) {
  const roadmaps = getRoadmaps(number);
  const badges = [];
  if (roadmaps.includes("Blind 75")) {
    badges.push("![Blind 75](https://img.shields.io/badge/Roadmap-Blind%2075-007acc?style=flat-square)");
  }
  if (roadmaps.includes("NeetCode 150")) {
    badges.push("![NeetCode 150](https://img.shields.io/badge/Roadmap-NeetCode%20150-6366f1?style=flat-square)");
  }
  return badges.join(" ");
}

export function getRoadmapProgress(syncedProblems = []) {
  const solvedSet = new Set(
    syncedProblems
      .map((p) => Number(p.number))
      .filter((n) => !Number.isNaN(n) && n > 0)
  );

  let blind75Solved = 0;
  for (const num of BLIND_75) {
    if (solvedSet.has(num)) blind75Solved++;
  }

  let neetcode150Solved = 0;
  for (const num of NEETCODE_150) {
    if (solvedSet.has(num)) neetcode150Solved++;
  }

  return {
    blind75: {
      solved: blind75Solved,
      total: BLIND_75.size,
      percent: Math.round((blind75Solved / BLIND_75.size) * 100)
    },
    neetcode150: {
      solved: neetcode150Solved,
      total: NEETCODE_150.size,
      percent: Math.round((neetcode150Solved / NEETCODE_150.size) * 100)
    }
  };
}
