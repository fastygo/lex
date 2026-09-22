import { sliceById, type SliceDef, type SliceQuestion } from "$lib/slices";

export const projectKind = "lex-flow-project";
export const projectVersion = 1;

export type ReasonCode =
  | "state-json"
  | "state-empty"
  | "questions"
  | "context"
  | "request-json"
  | "shape"
  | "trap"
  | "dwell"
  | "quiet"
  | "burst"
  | "lock"
  | "session";

export type DecisionDraft = {
  project_id: string;
  decision: { id: string; version: string };
  state: Record<string, unknown>;
  question_set: {
    id: string;
    version: string;
    questions: Record<string, SliceQuestion>;
  };
  context?: {
    pack: { id: string };
    snapshot: { id: string };
    pack_request: { id: string };
  };
};

export function questionBody(question: SliceQuestion): SliceQuestion {
  if (question.type === "choice") {
    return {
      id: question.id,
      type: "choice",
      instructions: question.instructions,
      options: question.options,
    };
  }
  if (question.type === "score") {
    return {
      id: question.id,
      type: "score",
      instructions: question.instructions,
      levels: question.levels,
    };
  }
  return { id: question.id, type: "noul", instructions: question.instructions };
}

export function contextStub(slice: SliceDef) {
  return {
    pack: { id: `pack_${slice.id}` },
    snapshot: { id: `snapshot_${slice.id}` },
    pack_request: { id: `request_${slice.id}` },
  };
}

export function buildDraft(slice: SliceDef, state: Record<string, unknown>, selected: string[]): DecisionDraft {
  const questions: Record<string, SliceQuestion> = {};
  for (const question of slice.questions) {
    if (selected.includes(question.id)) questions[question.id] = questionBody(question);
  }
  const draft: DecisionDraft = {
    project_id: "playground",
    decision: { id: slice.decisionId, version: "1" },
    state,
    question_set: { id: slice.questionSetId, version: "1", questions },
  };
  if (slice.context && selected.includes("context")) draft.context = contextStub(slice);
  return draft;
}

export function formatDraft(draft: DecisionDraft): string {
  return JSON.stringify(draft, null, 2);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function readDraft(text: string, slice: SliceDef): { draft: DecisionDraft | null; codes: ReasonCode[] } {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    return { draft: null, codes: ["request-json"] };
  }
  if (!isRecord(parsed) || !isRecord(parsed.decision) || !isRecord(parsed.question_set)) {
    return { draft: null, codes: ["shape"] };
  }
  if (!isRecord(parsed.state)) return { draft: null, codes: ["state-json"] };
  if (Object.keys(parsed.state).length === 0) return { draft: null, codes: ["state-empty"] };
  if (parsed.decision.id !== slice.decisionId || parsed.question_set.id !== slice.questionSetId) {
    return { draft: null, codes: ["shape"] };
  }
  const rawQuestions = parsed.question_set.questions;
  if (!isRecord(rawQuestions)) return { draft: null, codes: ["shape"] };

  const codes: ReasonCode[] = [];
  const known = new Set(slice.questions.map((question) => question.id));
  const selected = Object.keys(rawQuestions).filter((id) => known.has(id));
  if (selected.length !== slice.questions.length) codes.push("questions");
  for (const key of Object.keys(rawQuestions)) {
    if (!known.has(key)) codes.push("shape");
  }
  if (slice.context) {
    const context = parsed.context;
    const bound =
      isRecord(context) && isRecord(context.pack) && isRecord(context.snapshot) && isRecord(context.pack_request);
    if (!bound) codes.push("context");
  }

  const questions: Record<string, SliceQuestion> = {};
  for (const question of slice.questions) {
    if (!selected.includes(question.id)) continue;
    const raw = rawQuestions[question.id];
    if (!isRecord(raw) || raw.type !== question.type || typeof raw.instructions !== "string") {
      codes.push("shape");
      questions[question.id] = questionBody(question);
      continue;
    }
    questions[question.id] = {
      ...questionBody(question),
      instructions: raw.instructions,
    };
  }

  const draft: DecisionDraft = {
    project_id: "playground",
    decision: { id: slice.decisionId, version: "1" },
    state: parsed.state,
    question_set: { id: slice.questionSetId, version: "1", questions },
  };
  if (slice.context && isRecord(parsed.context)) {
    draft.context = contextStub(slice);
  }
  return { draft, codes: [...new Set(codes)] };
}

export function selectedFromDraft(slice: SliceDef, draft: DecisionDraft | null): string[] {
  if (!draft) return [];
  const ids = Object.keys(draft.question_set.questions);
  if (slice.context && draft.context) ids.push("context");
  return ids;
}

export function projectFile(sliceId: string, requestText: string) {
  return {
    kind: projectKind,
    version: projectVersion,
    sliceId: sliceById(sliceId).id,
    requestText,
  };
}

export function readProjectFile(value: unknown): { sliceId: string; requestText: string } | null {
  if (!isRecord(value)) return null;
  if (value.kind !== projectKind || value.version !== projectVersion) return null;
  if (typeof value.sliceId !== "string" || typeof value.requestText !== "string") return null;
  const slice = sliceById(value.sliceId);
  if (slice.id !== value.sliceId && !value.sliceId) return null;
  return { sliceId: slice.id, requestText: value.requestText };
}
