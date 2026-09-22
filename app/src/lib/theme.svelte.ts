const storageKey = "lex-theme";

function storedMode(): "light" | "dark" {
  const saved = localStorage.getItem(storageKey);
  if (saved === "light" || saved === "dark") return saved;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function applyTheme(mode: "light" | "dark") {
  document.documentElement.classList.toggle("dark", mode === "dark");
}

export function createTheme() {
  let mode = $state(storedMode());

  $effect(() => {
    applyTheme(mode);
    localStorage.setItem(storageKey, mode);
  });

  return {
    get mode() {
      return mode;
    },
    toggle() {
      mode = mode === "dark" ? "light" : "dark";
    },
  };
}
