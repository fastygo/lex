export type QuestionType = "noul" | "choice" | "score";

export type SliceQuestion = {
  id: string;
  type: QuestionType;
  instructions: string;
  options?: Record<string, string>;
  levels?: string[];
};

export type SliceDef = {
  id: string;
  title: string;
  domain: string;
  summary: string;
  decisionId: string;
  questionSetId: string;
  state: Record<string, unknown>;
  questions: SliceQuestion[];
  context: boolean;
};

const choice = (
  id: string,
  instructions: string,
  options: Record<string, string>,
): SliceQuestion => ({ id, type: "choice", instructions, options });

const noul = (id: string, instructions: string): SliceQuestion => ({
  id,
  type: "noul",
  instructions,
});

const score = (id: string, instructions: string, levels: string[]): SliceQuestion => ({
  id,
  type: "score",
  instructions,
  levels,
});

export const slices: SliceDef[] = [
  {
    id: "support-queue",
    title: "Support queue",
    domain: "Support",
    summary: "Separate order presence, queue, and urgency before anyone routes the ticket.",
    decisionId: "support-queue",
    questionSetId: "support.queue",
    state: { message: "Where is order 1042? It was due yesterday." },
    context: false,
    questions: [
      noul("mentions_order", "Does the message reference a specific order?"),
      choice("route", "Which queue should handle the message?", {
        shipping: "Delivery status.",
        billing: "Payments.",
        other: "None of these.",
        manual_review: "A human must decide.",
      }),
      score("urgency", "How urgent is the message?", ["low", "medium", "high"]),
    ],
  },
  {
    id: "inbound-intent",
    title: "Inbound intent",
    domain: "Product intake",
    summary: "Name the intent and whether a purchase flow is present. The next step stays outside LeX.",
    decisionId: "intent-step",
    questionSetId: "example.intent",
    state: { message: "I need a site to show recent client work." },
    context: false,
    questions: [
      choice("intent", "Which declared intent best matches the supplied state?", {
        portfolio: "Show a body of work.",
        catalogue: "Browse a product collection.",
        other: "None of the declared intents.",
      }),
      noul("has_purchase_flow", "Does the supplied state require a purchase flow?"),
    ],
  },
  {
    id: "refund-dispute",
    title: "Refund dispute",
    domain: "Billing",
    summary: "Detect a refund ask and a named amount, then choose an outcome that can stay with a person.",
    decisionId: "refund-step",
    questionSetId: "billing.refund",
    state: {
      message: "Please refund order 88. I was charged twice.",
      order_id: "88",
    },
    context: false,
    questions: [
      noul("asks_refund", "Does the message ask for money back?"),
      noul("names_amount", "Does the message name a concrete amount?"),
      choice("outcome", "Which handling outcome matches the message?", {
        refund_review: "A refund review is appropriate.",
        billing_question: "This is a billing question without a refund ask.",
        other: "None of these.",
        manual_review: "A human must decide.",
      }),
    ],
  },
  {
    id: "tool-gate",
    title: "Tool gate",
    domain: "Agents",
    summary: "Judge a prepared tool call. The slice records the judgment and does not execute the tool.",
    decisionId: "tool-gate",
    questionSetId: "agent.tool-gate",
    state: {
      tool: "issue_refund",
      arguments: { order_id: "88", amount: "20.00" },
    },
    context: false,
    questions: [
      noul("arguments_complete", "Are the supplied tool arguments complete for that tool?"),
      noul("irreversible", "Is the operation irreversible?"),
      choice("disposition", "How should the caller treat this prepared call?", {
        allow_ask: "The caller may ask for confirmation.",
        deny: "The caller should not proceed.",
        manual_review: "A human must decide.",
      }),
    ],
  },
  {
    id: "context-route",
    title: "Frozen context route",
    domain: "Evidence",
    summary: "Route from State while a frozen context binding stays on its own port.",
    decisionId: "context-binding-step",
    questionSetId: "example.context-binding",
    state: { task: "Classify the request using only the frozen context binding." },
    context: true,
    questions: [
      choice("route", "Which route is best supported by the supplied state?", {
        account_access: "Account access support.",
        billing: "Billing support.",
        other: "No declared route.",
      }),
      noul("has_frozen_context", "Is a frozen context binding supplied with this state?"),
    ],
  },
  {
    id: "requirement-rubric",
    title: "Requirement rubric",
    domain: "Design choice",
    summary: "Score a requirements object on an ordered scale and test one durability predicate.",
    decisionId: "storage-step",
    questionSetId: "example.storage",
    state: {
      requirements: { transactions: true, relational_queries: true, budget: "bounded" },
    },
    context: false,
    questions: [
      score(
        "relational_fit",
        "How strongly does the state fit the ordered relational-storage rubric?",
        ["weak", "adequate", "strong"],
      ),
      noul("requires_durability", "Does the supplied state require durable storage?"),
    ],
  },
  {
    id: "incident-class",
    title: "Incident class",
    domain: "Operations",
    summary: "Class a symptom, its severity, and whether people outside the team are affected.",
    decisionId: "incident-step",
    questionSetId: "ops.incident",
    state: {
      service: "checkout",
      symptom: "Payments fail for new sessions.",
      started: "2026-09-22T18:04:00Z",
    },
    context: false,
    questions: [
      score("severity", "How severe is the incident on this ordered scale?", [
        "low",
        "medium",
        "high",
      ]),
      noul("customer_facing", "Are people outside the operating team affected?"),
      choice("runbook", "Which runbook class matches the symptom?", {
        payments: "Payment path.",
        identity: "Sign-in or account access.",
        other: "None of these.",
        manual_review: "A human must decide.",
      }),
    ],
  },
  {
    id: "conflict-split",
    title: "Conflicting evidence",
    domain: "Review",
    summary: "Keep support, conflict, safety, and the action choice as separate blocks.",
    decisionId: "conflict-step",
    questionSetId: "review.conflict",
    state: {
      hypotheses: ["duplicate_charge", "single_charge"],
      note: "Two ledger lines share an order id and differ by amount.",
    },
    context: false,
    questions: [
      noul("supports_duplicate", "Does the state support a duplicate charge?"),
      noul("supports_single", "Does the state support a single charge?"),
      noul("evidence_conflicts", "Do the supplied facts conflict?"),
      noul("auto_action_safe", "Is an automatic action safe?"),
      choice("action", "Which next handling matches the state?", {
        hold: "Hold for review.",
        manual_review: "A human must decide.",
        other: "None of these.",
      }),
    ],
  },
  {
    id: "form-completeness",
    title: "Form completeness",
    domain: "Intake",
    summary: "One presence predicate per required field, plus a short completeness score.",
    decisionId: "intake-step",
    questionSetId: "intake.completeness",
    state: { name: "Ada", email: "", brief: "Redesign the account page." },
    context: false,
    questions: [
      noul("has_name", "Does the state include a name?"),
      noul("has_email", "Does the state include an email address?"),
      noul("has_brief", "Does the state include a brief?"),
      score("completeness", "How complete is the intake on this ordered scale?", [
        "missing",
        "partial",
        "complete",
      ]),
    ],
  },
  {
    id: "reply-policy",
    title: "Reply policy",
    domain: "Knowledge",
    summary: "Check a draft against a quoted rule before anyone treats it as sendable.",
    decisionId: "reply-policy",
    questionSetId: "support.reply-policy",
    state: {
      rule: "Refunds over 50 require a person.",
      draft: "I refunded 80 to your card.",
    },
    context: false,
    questions: [
      noul("draft_uses_rule", "Does the draft rely on the quoted rule?"),
      noul("unstated_promise", "Does the draft promise something the rule does not state?"),
      choice("disposition", "How should the caller treat the draft?", {
        send: "The draft matches the rule.",
        rewrite: "The draft should be rewritten.",
        manual_review: "A human must decide.",
      }),
    ],
  },
  {
    id: "product-abc",
    title: "Product ABC",
    domain: "Catalog",
    summary: "Judge history, band agreement, assortment role, and how strongly revenue share fits class A. The band stays in State.",
    decisionId: "product-abc",
    questionSetId: "catalog.abc",
    state: {
      sku: "SKU-1042",
      title: "USB-C cable 1 m",
      category: "Accessories",
      period: "2026-08",
      units_sold: 840,
      revenue: 420000,
      revenue_share: 0.18,
      margin_share: 0.04,
      abc_band: "A",
      periods_observed: 6,
    },
    context: false,
    questions: [
      noul("history_sufficient", "Does the state include enough observations to judge the product role?"),
      noul("band_matches_share", "Does the stated ABC band agree with the revenue share?"),
      choice("assortment_role", "Which assortment role best matches this product?", {
        traffic: "High revenue share with a small margin share. The product draws demand.",
        profit: "A material contribution to margin.",
        niche: "Small revenue share and small margin share.",
        other: "None of these roles.",
        manual_review: "A human must decide.",
      }),
      score("abc_strength", "How strongly does the revenue share fit class A on this ordered scale?", [
        "The share is small and does not look like class A.",
        "The share is noticeable, but class A is not clear from it.",
        "The share is large and typical of class A.",
      ]),
    ],
  },
];

export const firstSlice = slices[0];

export function sliceById(id: string): SliceDef {
  return slices.find((slice) => slice.id === id) ?? firstSlice;
}
