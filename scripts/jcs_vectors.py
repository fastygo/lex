#!/usr/bin/env python3
"""Generate conformance vectors using the independent, pinned rfc8785 package."""

import hashlib
import json
import pathlib
import sys

import rfc8785

ROOT = pathlib.Path(__file__).resolve().parents[1]
OUTPUT = ROOT / "internal" / "canonical" / "testdata" / "jcs-vectors.json"

ANCHORS = {
    '{"a":1,"b":2}': "43258cff783fe7036d8a43033f830adfc60ec037382473548ac742b888292777",
    '{"cafe":"caf\u00e9"}': "801a86b42bae9df69aecb1337a75a6988bcb5d6c0b54bb54c4425e3dc1514369",
    '{"name":"\u20ac"}': "080466493ecc711eb2010d0339912c06fcc8d7921c380aecbfb4f0b86ed18b69",
    '{"n":0}': "f3013f933b9fb80ab6d995e7ad9da36f683837ba1d81e950c943d40111eac2f0",
    '{"n":1}': "2bfd14f43d17fc7cea24e0917a8879b4b2f880b8baeec1b9d90fbaad655e71bd",
    '{"n":100}': "b39022c4ed96525c42cd0e7ce55308533962a655f1c19d5dac2f03e9dd995b2c",
    '{"n":1.23}': "c2f4a8099bdaf483ac3f465590b90ae2156f94d0d32c194bfbb06ca2289ad25f",
}


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate JSON key")
        result[key] = value
    return result


def parse_json(raw):
    return json.loads(raw, parse_int=float, parse_float=float, object_pairs_hook=unique_object)


def canonical(value):
    return rfc8785.dumps(value).decode("utf-8")


def digest(text):
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def jcs_vector(name, raw):
    form = canonical(parse_json(raw))
    return {"name": name, "kind": "jcs", "input": raw, "canonical": form, "sha256": digest(form)}


def source_vector(name, text):
    return {"name": name, "kind": "source_bytes", "input": text, "sha256": hashlib.sha256(text.encode("utf-8")).hexdigest()}


def reject_vector(name, raw):
    return {"name": name, "kind": "reject", "input": raw}


def bundle_digest(raw):
    value = parse_json(raw)
    if not isinstance(value, dict) or "bundle_hash" not in value:
        raise SystemExit("bundle_hash missing")
    del value["bundle_hash"]
    return digest(canonical(value))


def main():
    for form, expected in ANCHORS.items():
        actual = digest(form)
        if actual != expected:
            raise SystemExit(f"anchor mismatch for {form}: {actual}")
    vectors = [
        jcs_vector("key-order", '{"b":2,"a":1}'),
        jcs_vector("nested-key-order", '{"z":1,"a":[true,null]}'),
        jcs_vector("unicode-cafe", '{"cafe":"caf\u00e9"}'),
        jcs_vector("unicode-cafe-escaped", '{"cafe":"caf\\u00e9"}'),
        jcs_vector("unicode-euro", '{"name":"\\u20ac"}'),
        jcs_vector("negative-zero", '{"n":-0}'),
        jcs_vector("fractional-zero", '{"n":0.0}'),
        jcs_vector("negative-fractional-zero", '{"n":-0.0}'),
        jcs_vector("integral-float", '{"n":1.0}'),
        jcs_vector("small-exponent", '{"n":1e-7}'),
        jcs_vector("decimal-boundary", '{"n":1e-6}'),
        jcs_vector("large-decimal", '{"n":1e20}'),
        jcs_vector("large-exponent", '{"n":1e21}'),
        jcs_vector("utf16-property-order", '{"\\ufb33":1,"\\ud83d\\ude00":2}'),
        jcs_vector("exponent", '{"n":1e2}'),
        jcs_vector("trailing-fraction-zeros", '{"n":1.2300}'),
        jcs_vector("solidus-not-escaped", '{"url":"https://example.com/a"}'),
        jcs_vector("controls-escaped", '{"text":"line\\nnext"}'),
        jcs_vector("array-order", '{"items":["b","a"]}'),
        jcs_vector("without-self-hash", '{"id":"claim-1","protocol_version":"0.1"}'),
        jcs_vector("with-self-hash", '{"bundle_hash":"0000","id":"claim-1","protocol_version":"0.1"}'),
        source_vector("source-bytes", "The account is locked."),
        source_vector("source-bytes-one-edit", "The account is lockd."),
        source_vector("source-bytes-unicode", "caf\u00e9"),
        reject_vector("lone-surrogate", '{"text":"\\ud800"}'),
        reject_vector("duplicate-key", '{"id":"first","id":"second"}'),
        reject_vector("trailing-value", '{"id":"one"} {"id":"two"}'),
    ]
    for vector in vectors:
        if vector["kind"] == "jcs" and vector["canonical"] in ANCHORS:
            if vector["sha256"] != ANCHORS[vector["canonical"]]:
                raise SystemExit(f"vector {vector['name']} drifted from its anchor")
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT.write_text(json.dumps({"producer": "scripts/jcs_vectors.py", "vectors": vectors}, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    if "--bundle" in sys.argv:
        print(bundle_digest(sys.stdin.read()))
    else:
        main()
