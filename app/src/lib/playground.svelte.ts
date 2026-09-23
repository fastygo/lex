import { SvelteSet } from "svelte/reactivity";
import { copy } from "$lib/copy";
import { layoutSlice, legalLink, type FlowLink, type Placement, wiredIds } from "$lib/layout";
import {
  buildDraft,
  formatDraft,
  formatQuestions,
  projectFile,
  questionBody,
  readDraft,
  readProjectFile,
  readQuestions,
  readState,
  type QuestionsIssue,
  type ReasonCode,
  type StateIssue,
  type WireQuestion,
} from "$lib/request";
import { sliceById, type SliceDef } from "$lib/slices";

const dwellMs = 2000;
const quietMs = 1500;
const burstMs = 8000;
const burstWindowMs = 2000;
const burstCount = 6;
const runLockMs = 3500;
const sessionMs = 1200;

export function createPlayground(initialSliceId: string) {
  let sliceId = $state(sliceById(initialSliceId).id);
  let requestText = $state("");
  let stateText = $state("");
  let stateRevision = $state(0);
  let stateIssue = $state<StateIssue | null>(null);
  let questionBodies = $state<Record<string, WireQuestion>>({});
  let questionsText = $state("");
  let questionsRevision = $state(0);
  let questionsIssue = $state<QuestionsIssue | null>(null);
  let responseText = $state("{}");
  let links = $state<FlowLink[]>([]);
  let placements = $state<Record<string, Placement>>({});
  let revision = $state(0);
  let trap = $state("");
  let status = $state("");
  let now = $state(Date.now());
  let assembledAt = $state(0);
  let quietUntil = 0;
  let holdUntil = 0;
  let runReadyAt = 0;
  let mountedAt = Date.now();
  const clicks: number[] = [];

  function slice(): SliceDef {
    return sliceById(sliceId);
  }

  function structural(): ReasonCode[] {
    return readDraft(requestText, slice()).codes;
  }

  function linkCodes(): ReasonCode[] {
    const codes: ReasonCode[] = [];
    const wired = new SvelteSet(wiredIds(slice(), links));
    if (!slice().questions.every((question) => wired.has(question.id))) codes.push("questions");
    if (slice().context && !wired.has("context")) codes.push("context");
    if (!links.some((link) => link.source === "decision" && link.target === "output")) codes.push("output");
    return codes;
  }

  function blocking(): ReasonCode[] {
    return [...new SvelteSet<ReasonCode>([...structural(), ...linkCodes()])];
  }

  const errorCodes: ReasonCode[] = ["trap", "request-json", "shape", "state-json", "state-empty"];
  const waitCodes: ReasonCode[] = ["session", "dwell", "quiet", "burst", "lock"];

  function activeCodes(): ReasonCode[] {
    const codes = new SvelteSet<ReasonCode>(blocking());
    if (trap.trim() !== "") codes.add("trap");
    const t = now;
    if (codes.size === 0) {
      if (t < mountedAt + sessionMs) codes.add("session");
      if (assembledAt !== 0 && t < assembledAt + dwellMs) codes.add("dwell");
      if (t < quietUntil) codes.add("quiet");
      if (t < holdUntil) codes.add("burst");
      if (t < runReadyAt) codes.add("lock");
    }
    return [...codes];
  }

  function syncAssembly(codes: ReasonCode[]) {
    const ready = codes.length === 0 && trap.trim() === "";
    if (ready && assembledAt === 0) assembledAt = Date.now();
    if (!ready) assembledAt = 0;
  }

  function noteClick() {
    const t = Date.now();
    while (clicks.length > 0 && t - clicks[0] > burstWindowMs) clicks.shift();
    clicks.push(t);
    if (clicks.length >= burstCount) holdUntil = t + burstMs;
    quietUntil = t + quietMs;
  }

  function noteEdit() {
    quietUntil = Date.now() + quietMs;
  }

  function freshBodies(current: SliceDef): Record<string, WireQuestion> {
    const bodies: Record<string, WireQuestion> = {};
    for (const question of current.questions) bodies[question.id] = questionBody(question);
    return bodies;
  }

  function bodiesFromDraft(current: SliceDef, draft: ReturnType<typeof readDraft>["draft"]): Record<string, WireQuestion> {
    const bodies = freshBodies(current);
    if (!draft) return bodies;
    for (const question of current.questions) {
      const stored = draft.question_set.questions[question.id];
      if (stored) bodies[question.id] = stored;
    }
    return bodies;
  }

  function compose(state: Record<string, unknown>, nextLinks: FlowLink[]) {
    const next = buildDraft(slice(), state, wiredIds(slice(), nextLinks));
    for (const id of Object.keys(next.question_set.questions)) {
      const body = questionBodies[id];
      if (body) next.question_set.questions[id] = body;
    }
    return next;
  }

  function rebuild(nextLinks: FlowLink[]) {
    const parsed = readDraft(requestText, slice());
    const state = parsed.draft?.state ?? slice().state;
    requestText = formatDraft(compose(state, nextLinks));
    revision += 1;
  }

  function replaceStateBuffer(state: Record<string, unknown>) {
    stateText = JSON.stringify(state, null, 2);
    stateIssue = null;
    stateRevision += 1;
  }

  function replaceQuestionsBuffer(current: SliceDef, bodies: Record<string, WireQuestion>) {
    questionBodies = bodies;
    questionsText = formatQuestions(current, bodies);
    questionsIssue = null;
    questionsRevision += 1;
  }

  function publishState(state: Record<string, unknown>) {
    const formatted = formatDraft(compose(state, links));
    if (formatted === requestText) return;
    noteEdit();
    requestText = formatted;
    revision += 1;
    status = "";
    responseText = "{}";
    syncAssembly(blocking());
  }

  function publishQuestions(bodies: Record<string, WireQuestion>) {
    questionBodies = bodies;
    const parsed = readDraft(requestText, slice());
    const state = parsed.draft?.state ?? slice().state;
    const formatted = formatDraft(compose(state, links));
    noteEdit();
    status = "";
    responseText = "{}";
    if (formatted !== requestText) {
      requestText = formatted;
      revision += 1;
    }
    syncAssembly(blocking());
  }

  function writeFresh(id: string) {
    const next = sliceById(id);
    sliceId = next.id;
    links = [];
    placements = {};
    replaceQuestionsBuffer(next, freshBodies(next));
    requestText = formatDraft(compose(next.state, []));
    replaceStateBuffer(next.state);
    responseText = "{}";
    revision += 1;
    status = "";
    assembledAt = 0;
  }

  writeFresh(initialSliceId);
  revision = 0;

  return {
    get sliceId() {
      return sliceId;
    },
    get slice() {
      return slice();
    },
    get requestText() {
      return requestText;
    },
    get stateText() {
      return stateText;
    },
    get stateRevision() {
      return stateRevision;
    },
    get stateIssue() {
      return stateIssue;
    },
    get questionsText() {
      return questionsText;
    },
    get questionsRevision() {
      return questionsRevision;
    },
    get questionsIssue() {
      return questionsIssue;
    },
    get responseText() {
      return responseText;
    },
    get links() {
      return links;
    },
    get revision() {
      return revision;
    },
    get trap() {
      return trap;
    },
    set trap(value: string) {
      trap = value;
      syncAssembly(blocking());
    },
    get status() {
      return status;
    },
    get graph() {
      const current = slice();
      const questions = current.questions.map((question) => {
        const body = questionBodies[question.id];
        return body ? { ...question, instructions: body.instructions } : question;
      });
      return layoutSlice({ ...current, questions }, links, placements);
    },
    place(id: string, x: number, y: number) {
      if (id === "questions") return;
      const current = placements[id];
      if (current && current.x === x && current.y === y) return;
      placements = { ...placements, [id]: { x, y } };
    },
    reasons(): string[] {
      return activeCodes().map((code) => copy.playground.reasons[code]);
    },
    notice(): {
      variant: "default" | "destructive" | "success" | "warning";
      role: "status" | "alert";
      title: string;
      text: string;
    } {
      if (status) {
        return {
          variant: "success",
          role: "status",
          title: copy.playground.alertSuccess,
          text: status,
        };
      }
      const codes = activeCodes();
      const error = errorCodes.find((code) => codes.includes(code));
      if (error) {
        return {
          variant: "destructive",
          role: "alert",
          title: copy.playground.alertError,
          text: copy.playground.reasons[error],
        };
      }
      const wait = waitCodes.find((code) => codes.includes(code));
      if (wait) {
        return {
          variant: "warning",
          role: "status",
          title: slice().title,
          text: copy.playground.reasons[wait],
        };
      }
      const info = codes[0];
      return {
        variant: "default",
        role: "status",
        title: slice().title,
        text: info ? copy.playground.reasons[info] : copy.playground.alertReady,
      };
    },
    get canRun() {
      return this.reasons().length === 0;
    },
    tick() {
      now = Date.now();
      const codes = blocking();
      if (codes.length === 0 && trap.trim() === "" && assembledAt === 0) assembledAt = now;
      if (codes.length > 0 || trap.trim() !== "") assembledAt = 0;
    },
    load(id: string) {
      if (sliceId === sliceById(id).id && requestText) return;
      writeFresh(id);
    },
    forceLoad(id: string) {
      writeFresh(id);
    },
    connect(source: string | null, target: string | null) {
      if (!legalLink(slice(), source, target) || !source || !target) return;
      if (links.some((link) => link.source === source && link.target === target)) return;
      noteClick();
      const next = [...links, { source, target }];
      links = next;
      rebuild(next);
      status = "";
      syncAssembly(blocking());
    },
    removeLinks(removed: FlowLink[]) {
      if (removed.length === 0) return;
      const drop = new SvelteSet(removed.map((link) => `${link.source}->${link.target}`));
      const next = links.filter((link) => !drop.has(`${link.source}->${link.target}`));
      if (next.length === links.length) return;
      noteClick();
      links = next;
      rebuild(next);
      status = "";
      syncAssembly(blocking());
    },
    editState(text: string) {
      if (text === stateText) return;
      stateText = text;
      const read = readState(text);
      stateIssue = read.code;
      if (!read.state) return;
      publishState(read.state);
    },
    editQuestions(text: string) {
      if (text === questionsText) return;
      questionsText = text;
      const read = readQuestions(text, slice());
      questionsIssue = read.code;
      if (!read.bodies) return;
      publishQuestions(read.bodies);
    },
    run() {
      if (!this.canRun) return;
      const parsed = readDraft(requestText, slice());
      if (!parsed.draft || parsed.codes.length > 0 || linkCodes().length > 0) return;
      runReadyAt = Date.now() + runLockMs;
      status = copy.playground.composed;
      requestText = formatDraft(parsed.draft);
      responseText = "{}";
      revision += 1;
    },
    exportProject() {
      return projectFile(sliceId, requestText, links);
    },
    importProject(value: unknown): boolean {
      const file = readProjectFile(value);
      if (!file) return false;
      sliceId = file.sliceId;
      links = file.links;
      placements = {};
      requestText = file.requestText;
      const imported = readDraft(file.requestText, slice());
      replaceStateBuffer(imported.draft?.state ?? slice().state);
      replaceQuestionsBuffer(slice(), bodiesFromDraft(slice(), imported.draft));
      responseText = "{}";
      revision += 1;
      status = "";
      syncAssembly(blocking());
      return true;
    },
  };
}

export type Playground = ReturnType<typeof createPlayground>;
