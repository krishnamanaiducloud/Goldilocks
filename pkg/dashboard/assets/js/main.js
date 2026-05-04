// For scripts that should always be run on every page

import { setJavascriptAvailable } from "./utilities.js";

setJavascriptAvailable();

// --- Fix fragment links when <base> tag is set ---
// The <base> tag causes #anchor links to resolve against the base URL
// instead of the current page, breaking in-page navigation.
document.addEventListener("click", function (e) {
  const link = e.target.closest('a[href^="#"]');
  if (!link) return;
  const id = link.getAttribute("href").substring(1);
  const target = document.getElementById(id);
  if (target) {
    e.preventDefault();
    target.scrollIntoView({ behavior: "smooth" });
    // Update URL hash without navigation
    history.replaceState(null, "", "#" + id);
  }
});

// --- Dark mode toggle ---
function initTheme() {
  const saved = localStorage.getItem("goldilocks-theme");
  if (saved) {
    document.documentElement.setAttribute("data-theme", saved);
  }
}
initTheme();

window.toggleTheme = function () {
  const current = document.documentElement.getAttribute("data-theme");
  const isDark =
    current === "dark" ||
    (!current && window.matchMedia("(prefers-color-scheme: dark)").matches);
  const next = isDark ? "light" : "dark";
  document.documentElement.setAttribute("data-theme", next);
  localStorage.setItem("goldilocks-theme", next);
};

// --- Copy YAML to clipboard ---
window.copyYaml = function (btn) {
  const wrapper = btn.closest(".yaml-wrapper");
  const code = wrapper?.querySelector("code");
  if (!code) return;

  const text = code.textContent;
  navigator.clipboard.writeText(text).then(
    () => {
      btn.textContent = "Copied!";
      btn.classList.add("--copied");
      setTimeout(() => {
        btn.textContent = "Copy";
        btn.classList.remove("--copied");
      }, 2000);
    },
    () => {
      btn.textContent = "Failed";
      setTimeout(() => {
        btn.textContent = "Copy";
      }, 2000);
    }
  );
};
