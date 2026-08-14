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
        copyButton.textContent = 'Copiado ✓';
      } catch (_) {
        copyButton.textContent = 'Não foi possível copiar';
      }
      window.setTimeout(() => { copyButton.textContent = original; }, 1800);
    });
  });

  const stageTarget = document.querySelector('[data-stage-days]');
  if (stageTarget) {
    let sourceDate = stageTarget.dataset.projectUpdated || '';
    const events = [...document.querySelectorAll('[data-project-timeline] [data-event-date]')];
    const stageEvent = events.find((item) => {
      const type = (item.dataset.eventType || '').toLowerCase();
      return type.includes('andamento') || type.includes('status') || type.includes('fase');
    });
    if (stageEvent && stageEvent.dataset.eventDate) sourceDate = stageEvent.dataset.eventDate;
    const parsed = new Date(sourceDate);
    if (!Number.isNaN(parsed.getTime())) {
      const today = new Date();
      const diff = Math.max(0, Math.floor((today.getTime() - parsed.getTime()) / 86400000));
      stageTarget.textContent = diff === 0 ? 'Hoje' : `${diff} dia(s)`;
    } else {
      stageTarget.textContent = 'A definir';
    }
  }
})();
