/** Patterns lex-flow registers. Matches `package.json` → `ui8kit.aria`. */
export const ariaSubset = ["dialog", "disclosure", "tooltip", "alert"] as const;

export type AriaPattern = (typeof ariaSubset)[number];
