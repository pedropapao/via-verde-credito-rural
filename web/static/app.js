(() => {
  const btn = document.querySelector('[data-menu-toggle]');
  const sidebar = document.getElementById('sidebar');
  if (btn && sidebar) {
    btn.addEventListener('click', () => sidebar.classList.toggle('open'));
    document.addEventListener('click', (e) => {
      if (window.innerWidth <= 780 && sidebar.classList.contains('open') && !sidebar.contains(e.target) && e.target !== btn) sidebar.classList.remove('open');
    });
  }
})();
