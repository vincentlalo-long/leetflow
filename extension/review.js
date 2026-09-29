const GRADES = {
  again: { label: "Again", interval: 1, factor: 0.5 },
  hard: { label: "Hard", interval: 2, factor: 0.8 },
  good: { label: "Good", interval: 4, factor: 1.0 },
  easy: { label: "Easy", interval: 8, factor: 1.3 }
};

function dateOnly(date = new Date()) {
  return date.toISOString().slice(0, 10);
}

function addDays(date, days) {
  const result = new Date(date);
  result.setUTCDate(result.getUTCDate() + Math.max(1, Math.round(days)));
  return dateOnly(result);
}

export function ensureReview(problem) {
  if (problem.review) return problem;

  let initialStability = 2;
  // If struggled (took > 25 mins or failed multiple times) -> review sooner
  if ((problem.durationSeconds && problem.durationSeconds > 1500) || (problem.attemptsCount && problem.attemptsCount > 2)) {
    initialStability = 1;
  } else if (problem.durationSeconds && problem.durationSeconds < 600 && problem.attemptsCount === 1) {
    // Solved quickly in 1-shot -> review a bit later
    initialStability = 3;
  }

  return {
    ...problem,
    review: {
      due: addDays(new Date(), initialStability),
      stability: initialStability,
      reviews: 0,
      lastGrade: ""
    }
  };
}

export function dueReviews(problems, today = dateOnly()) {
  return problems.filter((problem) => {
    const review = ensureReview(problem).review;
    return review.due <= today;
  }).sort((a, b) => a.review.due.localeCompare(b.review.due));
}

export function gradeReview(problem, grade) {
  const definition = GRADES[grade] || GRADES.good;
  const current = ensureReview(problem);
  const stability = Math.max(1, current.review.stability * definition.factor + definition.interval / 2);
  return {
    ...current,
    review: {
      ...current.review,
      due: addDays(new Date(), stability),
      stability,
      reviews: current.review.reviews + 1,
      lastGrade: grade
    }
  };
}

export { GRADES, dateOnly };
