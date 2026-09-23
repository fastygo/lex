import type { Edge, Node } from "@xyflow/svelte";
import type { SliceDef } from "$lib/slices";

export type FlowLink = {
  source: string;
  target: string;
};

export type Placement = {
  x: number;
  y: number;
};

export type StepData = {
  title: string;
  detail: string;
  lane: "input" | "control" | "mechanism" | "output";
  included?: boolean;
  grouped?: boolean;
};

const cardWidth = 240;
const cardSlot = 128;
const rise = 96;
const columnGap = 28;
const pad = 16;

export function linkId(link: FlowLink): string {
  return `${link.source}->${link.target}`;
}

export function legalLink(slice: SliceDef, source: string | null, target: string | null): boolean {
  if (!source || !target || source === target) return false;
  if (source === "state" && (target === "questions" || (slice.context && target === "context"))) return true;
  if (target === "decision" && (source === "questions" || (slice.context && source === "context"))) return true;
  if (source === "decision" && target === "output") return true;
  return false;
}

export function wiredIds(slice: SliceDef, links: FlowLink[]): string[] {
  const selected: string[] = [];
  const groupLinked =
    links.some((link) => link.source === "state" && link.target === "questions") &&
    links.some((link) => link.source === "questions" && link.target === "decision");
  if (groupLinked) {
    for (const question of slice.questions) selected.push(question.id);
  }
  if (slice.context) {
    const fromState = links.some((link) => link.source === "state" && link.target === "context");
    const toDecision = links.some((link) => link.source === "context" && link.target === "decision");
    if (fromState && toDecision) selected.push("context");
  }
  return selected;
}

export function layoutSlice(
  slice: SliceDef,
  links: FlowLink[],
  placements: Record<string, Placement> = {},
): { nodes: Node[]; edges: Edge[] } {
  const wired = new Set(wiredIds(slice, links));
  const outputLinked = links.some((link) => link.source === "decision" && link.target === "output");
  const count = slice.questions.length;
  const groupWidth = pad * 2 + cardWidth * 2 + columnGap;
  const groupHeight = pad * 2 + cardSlot + Math.max(0, count - 1) * rise;
  const groupX = 48;
  const groupY = 128;
  const nodes: Node[] = [];
  const at = (id: string, x: number, y: number) => placements[id] ?? { x, y };

  const step = (id: string, x: number, y: number, data: StepData, parentId?: string): void => {
    nodes.push({
      id,
      type: "step",
      position: at(id, x, y),
      data,
      parentId,
      draggable: true,
      selectable: true,
      deletable: false,
      connectable: !parentId,
      zIndex: parentId ? 1 : 0,
    });
  };

  step("state", groupX + (groupWidth - cardWidth) / 2, 0, {
    title: "State",
    detail: "Input. Caller-owned JSON.",
    lane: "input",
    included: true,
  });

  nodes.push({
    id: "questions",
    type: "group",
    position: at("questions", groupX, groupY),
    data: {},
    width: groupWidth,
    height: groupHeight,
    style: `width: ${groupWidth}px; height: ${groupHeight}px; padding: 0;`,
    draggable: false,
    selectable: false,
    deletable: false,
    connectable: true,
  });

  slice.questions.forEach((question, index) => {
    const column = index % 2;
    step(
      question.id,
      pad + column * (cardWidth + columnGap),
      pad + index * rise,
      {
        title: question.id,
        detail: `${question.type}. ${question.instructions}`,
        lane: "control",
        included: wired.has(question.id),
        grouped: true,
      },
      "questions",
    );
  });

  if (slice.context) {
    step("context", groupX + groupWidth + 48, groupY + Math.max(0, groupHeight / 2 - 56), {
      title: "Context binding",
      detail: "Frozen pack, snapshot, and pack request.",
      lane: "control",
      included: wired.has("context"),
    });
  }

  const controlsReady = slice.questions.every((question) => wired.has(question.id)) && (!slice.context || wired.has("context"));
  const decisionY = groupY + groupHeight + 32;
  const centerX = groupX + (groupWidth - cardWidth) / 2;
  step("decision", centerX, decisionY, {
    title: "Decision",
    detail: "Mechanism. One POST /v1/decisions when Run opens.",
    lane: "mechanism",
    included: controlsReady,
  });
  step("output", centerX, decisionY + 112, {
    title: "DecisionSet",
    detail: outputLinked ? "Output. Answers, report, and replay bundle." : "Output waits until Decision is connected.",
    lane: "output",
    included: outputLinked,
  });

  const edges: Edge[] = links.filter((link) => legalLink(slice, link.source, link.target)).map((link) => ({
    id: linkId(link),
    source: link.source,
    target: link.target,
    interactionWidth: 20,
  }));

  return { nodes, edges };
}
