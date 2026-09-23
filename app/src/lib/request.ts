import { legalLink, type FlowLink } from "$lib/layout";
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
  | "session"
  | "output";

export type WireQuestion =
  | { type: "noul"; instructions: string }
  | { type: "choice"; instructions: string; options: Record<string, string> }
  | { type: "score"; instructions: string; levels: string[] };

export type DecisionDraft = {
  project_id: string;
  decision: { id: string; version: string };
  state: Record<string, unknown>;
  question_set: {
    id: string;
    version: string;
    questions: Record<string, WireQuestion>;
  };
  context?: {
    pack: { id: string };
    snapshot: { id: string };
    pack_request: { id: string };
  };
};

export function questionBody(question: SliceQuestion): WireQuestion {
  if (question.type === "choice") {
    return { type: "choice", instructions: question.instructions, options: question.options ?? {} };
  }
  if (question.type === "score") {
    return { type: "score", instructions: question.instructions, levels: question.levels ?? [] };
  }
  return { type: "noul", instructions: question.instructions };
}

export type JevQuestion =
  | { type: "noul"; instructions: string }
  | { type: "choice"; instructions: string; criteria: Record<string, string> }
  | { type: "score"; instructions: string; criteria: string[] };

export type JevRequest = {
  state: Record<string, unknown>;
  questions: Record<string, JevQuestion>;
};

export function jevQuestion(question: WireQuestion): JevQuestion {
  if (question.type === "choice") {
    return { type: "choice", instructions: question.instructions, criteria: question.options };
  }
  if (question.type === "score") {
    return { type: "score", instructions: question.instructions, criteria: question.levels };
  }
  return { type: "noul", instructions: question.instructions };
}

export function jevRequest(draft: DecisionDraft): JevRequest {
  const questions: Record<string, JevQuestion> = {};
  for (const [id, question] of Object.entries(draft.question_set.questions)) {
    questions[id] = jevQuestion(question);
  }
  return { state: draft.state, questions };
}

const choiceKey = /^[A-Za-z][A-Za-z0-9_-]{0,127}$/;

function isOptionMap(value: unknown): value is Record<string, string> {
  if (!isRecord(value)) return false;
  const keys = Object.keys(value);
  return keys.length >= 2 && keys.every((key) => choiceKey.test(key) && typeof value[key] === "string" && value[key].length > 0);
}

function isLevelList(value: unknown): value is string[] {
  if (!Array.isArray(value) || value.length < 2 || value.length > 10) return false;
  if (!value.every((item) => typeof item === "string" && item.length > 0)) return false;
  return new Set(value).size === value.length;
}

export function formatQuestions(slice: SliceDef, bodies: Record<string, WireQuestion>): string {
  const questions: Record<string, JevQuestion> = {};
  for (const question of slice.questions) {
    questions[question.id] = jevQuestion(bodies[question.id] ?? questionBody(question));
  }
  return JSON.stringify(questions, null, 2);
}

export type QuestionsIssue = "questions-json" | "questions-shape";

export function readQuestions(
  text: string,
  slice: SliceDef,
): { bodies: Record<string, WireQuestion> | null; code: QuestionsIssue | null } {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    return { bodies: null, code: "questions-json" };
  }
  if (!isRecord(parsed)) return { bodies: null, code: "questions-json" };
  const ids = slice.questions.map((question) => question.id);
  const keys = Object.keys(parsed);
  if (keys.length !== ids.length || ids.some((id) => !Object.prototype.hasOwnProperty.call(parsed, id))) {
    return { bodies: null, code: "questions-shape" };
  }
  const bodies: Record<string, WireQuestion> = {};
  for (const question of slice.questions) {
    const raw = parsed[question.id];
    const body = jevWire(question, raw);
    if (!body) return { bodies: null, code: "questions-shape" };
    bodies[question.id] = body;
  }
  return { bodies, code: null };
}

function storedWire(question: SliceQuestion, raw: unknown): WireQuestion | null {
  if (!isRecord(raw) || raw.type !== question.type || typeof raw.instructions !== "string" || raw.instructions.length === 0) {
    return null;
  }
  if (question.type === "choice") {
    if (!isOptionMap(raw.options)) return null;
    return { type: "choice", instructions: raw.instructions, options: raw.options };
  }
  if (question.type === "score") {
    if (!isLevelList(raw.levels)) return null;
    return { type: "score", instructions: raw.instructions, levels: raw.levels };
  }
  return { type: "noul", instructions: raw.instructions };
}

function jevWire(question: SliceQuestion, raw: unknown): WireQuestion | null {
  if (!isRecord(raw) || raw.type !== question.type || typeof raw.instructions !== "string" || raw.instructions.length === 0) {
    return null;
  }
  const fields = Object.keys(raw);
  if (question.type === "choice") {
    if (fields.some((field) => field !== "type" && field !== "instructions" && field !== "criteria")) return null;
    if (!isOptionMap(raw.criteria)) return null;
    return { type: "choice", instructions: raw.instructions, options: raw.criteria };
  }
  if (question.type === "score") {
    if (fields.some((field) => field !== "type" && field !== "instructions" && field !== "criteria")) return null;
    if (!isLevelList(raw.criteria)) return null;
    return { type: "score", instructions: raw.instructions, levels: raw.criteria };
  }
  if (fields.some((field) => field !== "type" && field !== "instructions")) return null;
  return { type: "noul", instructions: raw.instructions };
}

export function formatJev(draft: DecisionDraft): string {
  return JSON.stringify(jevRequest(draft), null, 2);
}

export function contextStub(slice: SliceDef) {
  return {
    pack: { id: `pack_${slice.id}` },
    snapshot: { id: `snapshot_${slice.id}` },
    pack_request: { id: `request_${slice.id}` },
  };
}

export function buildDraft(slice: SliceDef, state: Record<string, unknown>, selected: string[]): DecisionDraft {
  const questions: Record<string, WireQuestion> = {};
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

export type StateIssue = "state-json" | "state-empty";

export function readState(text: string): { state: Record<string, unknown> | null; code: StateIssue | null } {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    return { state: null, code: "state-json" };
  }
  if (!isRecord(parsed)) return { state: null, code: "state-json" };
  if (Object.keys(parsed).length === 0) return { state: null, code: "state-empty" };
  return { state: parsed, code: null };
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

  const questions: Record<string, WireQuestion> = {};
  for (const question of slice.questions) {
    if (!selected.includes(question.id)) continue;
    const raw = rawQuestions[question.id];
    const stored = storedWire(question, raw);
    if (!stored) {
      codes.push("shape");
      questions[question.id] = questionBody(question);
      continue;
    }
    questions[question.id] = stored;
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

export function readLinks(value: unknown, slice: SliceDef): FlowLink[] {
  if (!Array.isArray(value)) return [];
  const links: FlowLink[] = [];
  for (const item of value) {
    if (!isRecord(item) || typeof item.source !== "string" || typeof item.target !== "string") continue;
    if (!legalLink(slice, item.source, item.target)) continue;
    if (links.some((link) => link.source === item.source && link.target === item.target)) continue;
    links.push({ source: item.source, target: item.target });
  }
  return links;
}

export function projectFile(sliceId: string, requestText: string, links: FlowLink[]) {
  return {
    kind: projectKind,
    version: projectVersion,
    sliceId: sliceById(sliceId).id,
    requestText,
    links,
  };
}

export function readProjectFile(value: unknown): { sliceId: string; requestText: string; links: FlowLink[] } | null {
  if (!isRecord(value)) return null;
  if (value.kind !== projectKind || value.version !== projectVersion) return null;
  if (typeof value.sliceId !== "string" || typeof value.requestText !== "string") return null;
  const slice = sliceById(value.sliceId);
  if (slice.id !== value.sliceId && !value.sliceId) return null;
  return { sliceId: slice.id, requestText: value.requestText, links: readLinks(value.links, slice) };
}
