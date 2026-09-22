/** Host-independent opt-in diagnostics; safe in browsers, SSR and Node. */
export function isDevEnv(): boolean {
 return (globalThis as typeof globalThis & {__UI8KIT_DEV__?:boolean}).__UI8KIT_DEV__ === true;
}
