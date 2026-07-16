import { setJavascriptAvailable } from "./utilities.js";

setJavascriptAvailable();

const prefersDark = window.matchMedia("(prefers-color-scheme: dark)");
const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)");

function activeTheme() {
  return document.documentElement.dataset.theme || (prefersDark.matches ? "dark" : "light");
}

function updateThemeControl() {
  const button = document.querySelector("[data-theme-toggle]");
  if (!button) return;
  const dark = activeTheme() === "dark";
  button.setAttribute("aria-pressed", String(dark));
  button.setAttribute("aria-label", dark ? "Use light theme" : "Use dark theme");
  const label = button.querySelector(".theme-toggle__label");
  if (label) label.textContent = dark ? "Light theme" : "Dark theme";
}

function toggleTheme() {
  const next = activeTheme() === "dark" ? "light" : "dark";
  document.documentElement.dataset.theme = next;
  try {
    localStorage.setItem("goldilocks-theme", next);
  } catch {
    // Theme switching still works when browser storage is disabled.
  }
  updateThemeControl();
}

async function copyYaml(button) {
  const wrapper = button.closest(".yaml-wrapper");
  const code = wrapper?.querySelector("code");
  const status = wrapper?.querySelector("[data-copy-status]");
  if (!code || !status) return;

  try {
    await navigator.clipboard.writeText(code.textContent || "");
    button.textContent = "Copied";
    button.classList.add("--copied");
    status.textContent = "YAML copied to the clipboard.";
  } catch {
    button.textContent = "Copy failed";
    status.textContent = "YAML could not be copied. Select the code and copy it manually.";
  }

  window.setTimeout(() => {
    button.textContent = "Copy YAML";
    button.classList.remove("--copied");
  }, 2000);
}

document.addEventListener("click", (event) => {
  if (!(event.target instanceof Element)) return;

  const themeButton = event.target.closest("[data-theme-toggle]");
  if (themeButton) {
    toggleTheme();
    return;
  }

  const copyButton = event.target.closest("[data-copy-yaml]");
  if (copyButton instanceof HTMLButtonElement) {
    void copyYaml(copyButton);
    return;
  }

  const link = event.target.closest('a[href^="#"]');
  if (!link) return;
  const id = link.getAttribute("href")?.slice(1);
  const target = id ? document.getElementById(id) : null;
  if (!target) return;
  event.preventDefault();
  target.scrollIntoView({ behavior: reduceMotion.matches ? "auto" : "smooth" });
  history.replaceState(null, "", `#${id}`);
});

prefersDark.addEventListener("change", updateThemeControl);
updateThemeControl();
