// Applies the saved theme before first paint (external file so a strict CSP can forbid inline scripts).
(function () {
  try {
    var t = localStorage.getItem('fits-theme');
    var dark = t === 'dark' || ((t === null || t === 'system') && window.matchMedia('(prefers-color-scheme: dark)').matches);
    if (dark) document.documentElement.classList.add('dark');
  } catch (e) {}
})();
