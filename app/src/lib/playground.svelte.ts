import { copy } from "$lib/copy";
import { layoutSlice, legalLink, type FlowLink, type Placement, wiredIds } from "$lib/layout";
import {
  buildDraft,
  formatDraft,
  projectFile,
  readDraft,
  readProjectFile,
  type ReasonCode,
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
    const wired = new Set(wiredIds(slice(), links));
    if (!slice().questions.every((question) => wired.has(question.id))) codes.push("questions");
    if (slice().context && !wired.has("context")) codes.push("context");
    if (!links.some((link) => link.source === "decision" && link.target === "output")) codes.push("output");
    return codes;
  }

  function blocking(): ReasonCode[] {
    return [...new Set<ReasonCode>([...structural(), ...linkCodes()])];
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

  function rebuild(nextLinks: FlowLink[]) {
    const parsed = readDraft(requestText, slice());
    const state = parsed.draft?.state ?? slice().state;
    const next = buildDraft(slice(), state, wiredIds(slice(), nextLinks));
    if (parsed.draft) {
      for (const [key, question] of Object.entries(parsed.draft.question_set.questions)) {
        if (next.question_set.questions[key]) next.question_set.questions[key] = question;
      }
    }
    requestText = formatDraft(next);
    revision += 1;
  }

  function writeFresh(id: string) {
    const next = sliceById(id);
    sliceId = next.id;
    links = [];
    placements = {};
    requestText = formatDraft(buildDraft(next, next.state, []));
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
      return layoutSlice(slice(), links, placements);
    },
    place(id: string, x: number, y: number) {
      if (id === "questions") return;
      const current = placements[id];
      if (current && current.x === x && current.y === y) return;
      placements = { ...placements, [id]: { x, y } };
    },
    reasons(): string[] {
      const codes = new Set<ReasonCode>(blocking());
      if (trap.trim() !== "") codes.add("trap");
      const t = now;
      if (codes.size === 0) {
        if (t < mountedAt + sessionMs) codes.add("session");
        if (assembledAt !== 0 && t < assembledAt + dwellMs) codes.add("dwell");
        if (t < quietUntil) codes.add("quiet");
        if (t < holdUntil) codes.add("burst");
        if (t < runReadyAt) codes.add("lock");
      }
      return [...codes].map((code) => copy.playground.reasons[code]);
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
      const drop = new Set(removed.map((link) => `${link.source}->${link.target}`));
      const next = links.filter((link) => !drop.has(`${link.source}->${link.target}`));
      if (next.length === links.length) return;
      noteClick();
      links = next;
      rebuild(next);
      status = "";
      syncAssembly(blocking());
    },
    editRequest(text: string) {
      if (text === requestText) return;
      noteEdit();
      requestText = text;
      status = "";
      syncAssembly(blocking());
    },
    run() {
      if (!this.canRun) return;
      const parsed = readDraft(requestText, slice());
      if (!parsed.draft || parsed.codes.length > 0 || linkCodes().length > 0) return;
      runReadyAt = Date.now() + runLockMs;
      status = copy.playground.composed;
      requestText = formatDraft(parsed.draft);
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
      revision += 1;
      status = "";
      syncAssembly(blocking());
      return true;
    },
  };
}

export type Playground = ReturnType<typeof createPlayground>;
