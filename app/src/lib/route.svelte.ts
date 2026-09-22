import { firstSlice, sliceById } from "$lib/slices";

export type AppRoute =
  | { name: "home" }
  | { name: "slices" }
  | { name: "playground"; sliceId: string };

export function parseRoute(pathname: string): AppRoute {
  if (pathname === "/slices") return { name: "slices" };
  if (pathname === "/playground") return { name: "playground", sliceId: firstSlice.id };
  if (pathname.startsWith("/playground/")) {
    const id = decodeURIComponent(pathname.slice("/playground/".length).replace(/\/$/, ""));
    return { name: "playground", sliceId: sliceById(id).id };
  }
  return { name: "home" };
}

export function createRouter() {
  let pathname = $state(window.location.pathname);

  $effect(() => {
    const onPop = () => {
      pathname = window.location.pathname;
    };
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  });

  function navigate(href: string) {
    if (window.location.pathname === href) return;
    window.history.pushState({}, "", href);
    pathname = href;
  }

  return {
    get pathname() {
      return pathname;
    },
    get route(): AppRoute {
      return parseRoute(pathname);
    },
    navigate,
  };
}
