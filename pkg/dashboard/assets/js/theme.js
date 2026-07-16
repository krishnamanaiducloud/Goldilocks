(() => {
  try {
    const savedTheme = localStorage.getItem("goldilocks-theme");
    if (savedTheme === "dark" || savedTheme === "light") {
      document.documentElement.dataset.theme = savedTheme;
    }
  } catch {
    // Storage can be disabled by browser policy; system preference remains the fallback.
  }
})();
