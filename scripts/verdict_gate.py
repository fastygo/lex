#!/usr/bin/env python3
"""Independent claim-validation 0.2.0 verdict reader.

This script does not import the Go module. It repeats the pinned threshold
rules and the inference-only evidence rule, then applies the same precedence.
It does not rebuild a Context pack.
"""

import json
import sys

SUPPORT_MIN = 0.7
ESTABLISH_MIN = 0.8
REFUTE_MIN = 0.8
CONFLICT_MIN = 0.5
SAFETY_MIN = 0.8
NOULS = ("support", "established", "refuted", "conflict", "safe_to_auto_act")
CHOICES = ("proceed", "reject", "manual_review", "other")
RANK = {
    "error": 6,
    "conflict": 5,
    "insufficient": 4,
    "manual_review": 3,
    "rejected": 2,
    "validated": 1,
}


def probability(value):
    return isinstance(value, (int, float)) and not isinstance(value, bool) and 0 <= value <= 1


def answers_of(bundle):
    decision = bundle.get("decision_set")
    if not isinstance(decision, dict):
        return {}
    answers = decision.get("answers")
    if not isinstance(answers, dict):
        return {}
    return answers


def validation_error(answers):
    for name in NOULS:
        answer = answers.get(name)
        if not isinstance(answer, dict) or answer.get("type") != "noul" or not probability(answer.get("noul")):
            return True
    action = answers.get("action")
    if not isinstance(action, dict) or action.get("type") != "choice":
        return True
    choice = action.get("choice")
    probabilities = action.get("probabilities")
    if choice not in CHOICES or not isinstance(probabilities, dict) or set(probabilities) != set(CHOICES):
        return True
    total = 0.0
    for name in CHOICES:
        value = probabilities[name]
        if not probability(value):
            return True
        total += float(value)
    if abs(total - 1) > 1e-9:
        return True
    selected = float(probabilities[choice])
    if any(float(value) > selected for value in probabilities.values()):
        return True
    return False


def noul(answers, name):
    return float(answers[name]["noul"])


def inference_only(bundle):
    context = bundle.get("context")
    if not isinstance(context, dict):
        return False
    pack = context.get("pack")
    if not isinstance(pack, dict):
        return False
    items = pack.get("evidence_items")
    if not isinstance(items, list):
        return False
    admissible = 0
    inference = 0
    for item in items:
        if not isinstance(item, dict):
            continue
        if item.get("class") == "model_inference":
            inference += 1
        elif item.get("class") == "source_text" and item.get("trust_level") == "project":
            admissible += 1
    return admissible == 0 and inference > 0


def policy_findings(answers):
    support = noul(answers, "support")
    established = noul(answers, "established")
    refuted = noul(answers, "refuted")
    conflict = noul(answers, "conflict")
    safety = noul(answers, "safe_to_auto_act")
    action = answers["action"]["choice"]
    findings = []
    negative = refuted >= REFUTE_MIN
    contradictory = negative and (established >= ESTABLISH_MIN or support >= SUPPORT_MIN)
    if conflict >= CONFLICT_MIN or contradictory:
        findings.append("conflict")
    if support < SUPPORT_MIN and not negative:
        findings.append("insufficient")
    if established < ESTABLISH_MIN and not negative:
        findings.append("insufficient")
    if negative:
        findings.append("rejected")
    if action in ("manual_review", "other"):
        findings.append("manual_review")
    elif action == "proceed":
        if safety < SAFETY_MIN:
            findings.append("manual_review")
        if support < SUPPORT_MIN or established < ESTABLISH_MIN or conflict >= CONFLICT_MIN or negative:
            findings.append("manual_review")
    elif action == "reject" and (not negative or established >= ESTABLISH_MIN):
        findings.append("manual_review")
    return findings


def verdict(bundle):
    parsed = answers_of(bundle)
    if validation_error(parsed):
        return "error"
    findings = policy_findings(parsed)
    if inference_only(bundle):
        findings.append("insufficient")
    selected = "validated"
    for finding in findings:
        if RANK[finding] > RANK[selected]:
            selected = finding
    return selected


def main():
    bundle = json.loads(sys.stdin.read())
    print(verdict(bundle))


if __name__ == "__main__":
    main()
