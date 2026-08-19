(() => {
  const copyText = async (text) => {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      return;
    }
    const area = document.createElement('textarea');
    area.value = text;
    area.style.position = 'fixed';
    area.style.opacity = '0';
    document.body.appendChild(area);
    area.focus();
    area.select();
    document.execCommand('copy');
    area.remove();
  };

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
        await copyText(text);
        copyButton.textContent = 'Copiado ✓';
      } catch (_) {
        copyButton.textContent = 'Não foi possível copiar';
      }
      window.setTimeout(() => { copyButton.textContent = original; }, 1800);
    });
  });

  document.querySelectorAll('[data-copy-open]').forEach((openButton) => {
    openButton.addEventListener('click', () => {
      const text = openButton.dataset.copyText || '';
      const url = openButton.dataset.openUrl || '';
      const original = openButton.textContent;
      if (text) {
        copyText(text).then(() => {
          openButton.textContent = 'CAR copiado ✓';
          window.setTimeout(() => { openButton.textContent = original; }, 1800);
        }).catch(() => {});
      }
      if (url) window.open(url, '_blank', 'noopener');
    });
  });

  document.querySelectorAll('[data-checklist-autosubmit]').forEach((select) => {
    select.addEventListener('change', () => {
      const form = select.closest('form');
      if (!form || select.dataset.saving === '1') return;
      select.dataset.saving = '1';
      select.setAttribute('aria-busy', 'true');
      const saveButton = form.querySelector('button[type="submit"]');
      if (saveButton) {
        saveButton.disabled = true;
        saveButton.textContent = 'Salvando...';
      }
      // Não desabilitar o select: campos disabled não são enviados no POST.
      form.requestSubmit();
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

  const receiptHelper = document.querySelector('[data-receipt-helper]');
  if (receiptHelper) {
    const cpfInput = receiptHelper.querySelector('[data-representative-cpf]');
    const buildButton = receiptHelper.querySelector('[data-build-receipt-request]');
    const requestBox = receiptHelper.querySelector('[data-receipt-request-box]');
    const messageField = receiptHelper.querySelector('[data-receipt-message]');
    const whatsappButton = receiptHelper.querySelector('[data-open-whatsapp]');
    const stateEmailButton = receiptHelper.querySelector('[data-state-receipt-email]');
    const carNumber = receiptHelper.dataset.carNumber || '';

    const onlyDigits = (value) => (value || '').replace(/\D/g, '').slice(0, 11);
    const formatCPF = (value) => {
      const digits = onlyDigits(value);
      if (digits.length <= 3) return digits;
      if (digits.length <= 6) return `${digits.slice(0, 3)}.${digits.slice(3)}`;
      if (digits.length <= 9) return `${digits.slice(0, 3)}.${digits.slice(3, 6)}.${digits.slice(6)}`;
      return `${digits.slice(0, 3)}.${digits.slice(3, 6)}.${digits.slice(6, 9)}-${digits.slice(9)}`;
    };
    const validCPF = (value) => {
      const cpf = onlyDigits(value);
      if (cpf.length !== 11 || /^(\d)\1{10}$/.test(cpf)) return false;
      const calc = (length) => {
        let sum = 0;
        for (let i = 0; i < length; i += 1) sum += Number(cpf[i]) * (length + 1 - i);
        const digit = (sum * 10) % 11;
        return digit === 10 ? 0 : digit;
      };
      return calc(9) === Number(cpf[9]) && calc(10) === Number(cpf[10]);
    };

    if (cpfInput) {
      cpfInput.addEventListener('input', () => {
        cpfInput.value = formatCPF(cpfInput.value);
        cpfInput.removeAttribute('aria-invalid');
      });
    }

    if (buildButton && cpfInput && requestBox && messageField) {
      buildButton.addEventListener('click', () => {
        const cpf = formatCPF(cpfInput.value);
        const original = buildButton.textContent;
        if (!validCPF(cpf)) {
          cpfInput.setAttribute('aria-invalid', 'true');
          cpfInput.focus();
          buildButton.textContent = 'Confira o CPF';
          window.setTimeout(() => { buildButton.textContent = original; }, 1800);
          return;
        }

        messageField.value = `Olá! Preciso acessar o seu CAR para baixar a segunda via do Recibo de Inscrição e o Demonstrativo, sem utilizar ou pedir a sua senha GOV.BR.\n\nPor favor:\n1. Acesse https://www.car.gov.br/#/central/acesso\n2. Entre com a sua própria conta GOV.BR.\n3. Abra a opção “Gerenciar Vínculos”.\n4. Escolha “Vincular Representante”.\n5. Informe meu CPF: ${cpf}\n6. Vincule o representante ao CAR: ${carNumber}\n\nDepois me avise que o vínculo foi concluído. Eu acessarei a Central com a minha própria conta GOV.BR para obter o Recibo e o Demonstrativo. Você poderá remover esse vínculo posteriormente na própria Central.\n\nImportante: não preciso da sua senha GOV.BR.`;
        requestBox.hidden = false;
        requestBox.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      });
    }

    if (whatsappButton && messageField) {
      whatsappButton.addEventListener('click', () => {
        const text = (messageField.value || '').trim();
        if (!text) return;
        window.open(`https://wa.me/?text=${encodeURIComponent(text)}`, '_blank', 'noopener');
      });
    }

    if (stateEmailButton) {
      stateEmailButton.addEventListener('click', () => {
        const email = stateEmailButton.dataset.email || '';
        if (!email) return;
        const subject = `Solicitação de cópia do Recibo de Inscrição do CAR - ${carNumber}`;
        const body = `Prezados,\n\nSolicito orientação e, se possível, a cópia do Recibo de Inscrição do Cadastro Ambiental Rural referente ao imóvel abaixo:\n\nCAR: ${carNumber}\n\nCaso sejam necessários documentos de identificação, autorização do titular ou outros comprovantes, por favor informem os documentos e o procedimento correto para atendimento.\n\nAtenciosamente,`;
        window.location.href = `mailto:${email}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`;
      });
    }
  }
})();
