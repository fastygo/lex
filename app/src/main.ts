import { mount } from "svelte";
import App from "./App.svelte";
import "./assets/styles/styles.css";

const target = document.getElementById("app");
if (!target) {
  throw new Error("missing #app");
}

mount(App, { target });
