# What LeX can ask Jev

Status: canary release. The generic typed-decision API uses envelope `0.2`;
deprecated legacy claim-validation uses envelope `0.1`. This file maps the research question
maps under [examples/](examples/) onto both surfaces. It is not a second
protocol.

`POST /v1/decisions` accepts agent-owned JSON `state` and an agent-owned
QuestionSet. LeX maps Noul, Choice, and Score definitions to Jev, validates the
raw answer contract, and returns a replayable DecisionSet. The agent
interprets it and decides the next step. Optional frozen Context is a binding,
not automatic state.

`POST /v1/evaluations` is deprecated compatibility behavior. It takes one
claim (`query`), one evidence input (`sources` or `frozen_context`), and the
embedded profile. The deployment asks six fixed questions and derives a
semantic verdict.

The fixed questions are five independent Noul predicates (`support`,
`established`, `refuted`, `conflict`, `safe_to_auto_act`) and one Choice
(`action`: `proceed`, `reject`, `manual_review`, `other`). The direct adapter
on the canary is `direct-systemone` / `jev-1.13.0`. Retrieval is exact phrase
on the claim text. Replay repeats the verdict from the sealed bundle and does
not call Jev again.

The JSON maps in `examples/*/questions-*.json` stay research fixtures. Sending
one directly to the legacy evaluation route is rejected before a provider call.
An agent can translate the same state and question meanings into a generic
decision request. The 19 bodies in
[test-vercel/requests/](test-vercel/requests/) are legacy scenarios rewritten
as single claims.

## Already callable through `/v1/decisions`

- [x] Caller-defined Noul, Choice, and Score questions over arbitrary JSON
  state, up to 64 questions.
- [x] Choice options and ordered Score levels sent unchanged in meaning to the
  Jev adapter; raw typed answers and resolved model returned.
- [x] Arbitrary agent decision identity, state hash, QuestionSet hash, adapter
  pin, and model pin sealed in a replay bundle.
- [x] Optional caller-frozen Context state verified as a binding without being
  merged into state.
- [x] Structural replay without a second Jev or retrieval call.

## Already callable through legacy `/v1/evaluations`

- [x] One claim over caller text. LeX freezes it with Context `memory-exact-v1` and asks the six profile questions. Empty exact selection returns `insufficient` and does not call Jev.
- [x] The same claim over a caller-frozen Context state (`frozen_context`). Context must reproduce the pack before Jev is called.
- [x] Playground claims whose phrase occurs in `examples/playground/typesafe-jev.md`: `structured decisions`, `cannot generate string responses`, `no separate confidence field`, `evaluated independently and in parallel`.
- [x] LLM claims whose phrase occurs in `examples/llm/llm.md`: `next token`, `Hallucination is structural`, `Temperature 0`, `treat the model output as a proposal`.
- [x] Context-scenario claims: `account access`, `restore access`, `refund`, `duplicate charge`, including the frozen pack for `account access`.
- [x] Chaos claims, one case per request: raw and augmented `cannot sign in`; resolved `cannot sign in` and `account access`; conflicted `cannot sign in` and `duplicate charge`.
- [x] Noul and Choice answers inside the profile. The verifier keeps raw answers, applies the uncalibrated thresholds, and returns `validated`, `rejected`, `insufficient`, `conflict`, `manual_review`, or `error`.
- [x] Replay of each of those sealed bundles on the deployment, without a second Jev call.

These calls check whether the frozen text supports the stated claim. They do
not reproduce the original Jev question ids.

## Not yet

- [x] A caller-chosen QuestionSet on `/v1/decisions`. The legacy profile
  registry still has one compatibility profile.
- [x] Score on `/v1/decisions`. `claim-validation` `0.2.0` still does not ask
  a Score.
- [x] A domain Choice such as `billing` / `account_access` / `other` on
  `/v1/decisions`; the vocabulary remains caller-owned.
- [ ] Several cases in one evaluation. Chaos is four representations; LeX evaluates one claim and one frozen state per request.
- [ ] A second Jev call whose questions depend on the first answer (fan-out, cascades, "fetch more evidence, then ask").
- [ ] The hosted adapter on this deployment. Local fixtures pass. The canary resolves `direct-systemone` only.
- [ ] Thresholds justified by a calibration report. The policy stays `uncalibrated`.

## Refuse

Keep these out of the protocol. They are either a direct Jev call, a Context
mechanism, or an authority LeX does not hold.

- [x] Refuse the raw maps `questions-jev.json`, `questions-llm.json`, `questions-context.json`, and `questions-chaos.json` as evaluation bodies. They are not claims, and they would let the caller define the verdict.
- [x] Refuse caller-supplied instructions, criteria, or a question id on legacy
  `POST /v1/evaluations`. Generic `/v1/decisions` deliberately accepts
  caller-owned questions, but does not interpret or authorize them.
- [x] Refuse using one Choice both to name a route and to authorize an action. Support stays a Noul; the action Choice is not permission.
- [x] Refuse Score inside `claim-validation` until a second, unrelated profile needs an ordered rubric.
- [x] Refuse chat, string generation, summarization, and extraction of a value the evidence does not already contain. Jev does not return prose, and LeX does not ask it to.
- [x] Refuse counting, arithmetic, and date ordering as LeX questions. The playground note already treats those as unreliable for Jev.
- [x] Refuse retrieval, reranking, and pack construction inside the decision call. Context already did that before the questions are asked.
- [x] Refuse treating a Jev probability, a high confidence, or an LLM sentence in the pack as authority for a refund, a route, or any other side effect.
