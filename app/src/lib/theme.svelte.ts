function systemMode(): "light" | "dark" {
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function dropStoredTheme() {
  try {
    localStorage.removeItem("lex-theme");
  } catch {
    // Storage can throw in a locked browser. The theme still stays in memory.
  }
}

export function applyTheme(mode: "light" | "dark") {
  document.documentElement.classList.toggle("dark", mode === "dark");
}

export function createTheme() {
  dropStoredTheme();
  const initial = systemMode();
  let mode = $state(initial);
  applyTheme(initial);

  return {
    get mode() {
      return mode;
    },
    toggle() {
      mode = mode === "dark" ? "light" : "dark";
      applyTheme(mode);
    },
  };
}
