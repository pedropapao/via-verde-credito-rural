(() => {
  const btn = document.querySelector('[data-menu-toggle]');
  const sidebar = document.getElementById('sidebar');
  if (btn && sidebar) {
    btn.addEventListener('click', () => sidebar.classList.toggle('open'));
    document.addEventListener('click', (e) => {
      if (window.innerWidth <= 780 && sidebar.classList.contains('open') && !sidebar.contains(e.target) && e.target !== btn) sidebar.classList.remove('open');
    });
  }

  document.querySelectorAll('[data-copy-target]').forEach((copyButton) => {
    copyButton.addEventListener('click', async () => {
      const target = document.querySelector(copyButton.dataset.copyTarget);
      if (!target) return;
      const text = typeof target.value === 'string' ? target.value : target.textContent;
      const original = copyButton.textContent;
      try {
        if (navigator.clipboard && window.isSecureContext) {
          await navigator.clipboard.writeText(text);
        } else {
          const area = document.createElement('textarea');
          area.value = text;
          area.style.position = 'fixed';
          area.style.opacity = '0';
          document.body.appendChild(area);
          area.focus();
          area.select();
          document.execCommand('copy');
          area.remove();
        }
        copyButton.textContent = 'Resumo copiado ✓';
      } catch (_) {
        copyButton.textContent = 'Não foi possível copiar';
      }
      window.setTimeout(() => { copyButton.textContent = original; }, 1800);
    });
  });
})();
