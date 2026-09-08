(() => {
  const PDFJS_VERSION = '6.3.289';
  const TESSERACT_VERSION = '7.0.0';
  const MAX_PDF_PAGES = 30;
  const MIN_NATIVE_TEXT = 80;
  const MIN_OCR_TEXT = 20;

  const form = document.querySelector('form[action="/autoproject/analyze"]');
  if (!form) return;
  const fileInput = form.querySelector('input[type="file"][name="files"]');
  const payloadInput = form.querySelector('input[name="ocr_payload"]');
  const status = form.querySelector('[data-ocr-status]');
  const submitButton = form.querySelector('button[type="submit"]');
  if (!fileInput || !payloadInput || !submitButton) return;

  const setStatus = (message, mode = '') => {
    if (!status) return;
    status.textContent = message;
    status.dataset.mode = mode;
    status.hidden = !message;
  };

  const loadClassicScript = (src) => new Promise((resolve, reject) => {
    const existing = document.querySelector(`script[data-autoproject-src="${src}"]`);
    if (existing) {
      if (existing.dataset.loaded === '1') resolve();
      else {
        existing.addEventListener('load', resolve, { once: true });
        existing.addEventListener('error', reject, { once: true });
      }
      return;
    }
    const script = document.createElement('script');
    script.src = src;
    script.async = true;
    script.dataset.autoprojectSrc = src;
    script.addEventListener('load', () => {
      script.dataset.loaded = '1';
      resolve();
    }, { once: true });
    script.addEventListener('error', reject, { once: true });
    document.head.appendChild(script);
  });

  const nativeTextForPage = async (page) => {
    try {
      const content = await page.getTextContent();
      return content.items.map((item) => item && item.str ? item.str : '').join(' ').replace(/\s+/g, ' ').trim();
    } catch (_) {
      return '';
    }
  };

  const renderPage = async (page) => {
    const base = page.getViewport({ scale: 1 });
    let scale = 1.8;
    if (base.width * scale > 2400) scale = 2400 / base.width;
    if (base.height * scale > 3200) scale = Math.min(scale, 3200 / base.height);
    scale = Math.max(scale, 1.15);
    const viewport = page.getViewport({ scale });
    const canvas = document.createElement('canvas');
    canvas.width = Math.ceil(viewport.width);
    canvas.height = Math.ceil(viewport.height);
    const ctx = canvas.getContext('2d', { alpha: false, willReadFrequently: true });
    ctx.fillStyle = '#fff';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    await page.render({ canvasContext: ctx, viewport }).promise;
    return canvas;
  };

  const recognizeScannedPDFs = async (files) => {
    const pdfFiles = files.filter((file) => file && file.name && file.name.toLowerCase().endsWith('.pdf'));
    if (!pdfFiles.length) return [];

    setStatus('Preparando leitura de PDFs escaneados…', 'working');
    const pdfjsLib = await import(`https://cdn.jsdelivr.net/npm/pdfjs-dist@${PDFJS_VERSION}/build/pdf.min.mjs`);
    pdfjsLib.GlobalWorkerOptions.workerSrc = `https://cdn.jsdelivr.net/npm/pdfjs-dist@${PDFJS_VERSION}/build/pdf.worker.min.mjs`;

    let tesseractReady = false;
    let worker = null;
    const results = [];

    try {
      for (let fileIndex = 0; fileIndex < pdfFiles.length; fileIndex++) {
        const file = pdfFiles[fileIndex];
        setStatus(`Verificando ${file.name} (${fileIndex + 1}/${pdfFiles.length})…`, 'working');
        const bytes = new Uint8Array(await file.arrayBuffer());
        const loadingTask = pdfjsLib.getDocument({ data: bytes, isEvalSupported: false, useSystemFonts: true });
        const pdf = await loadingTask.promise;
        const pagesToCheck = Math.min(pdf.numPages, MAX_PDF_PAGES);
        const pieces = [];
        let ocrPages = 0;

        for (let pageNo = 1; pageNo <= pagesToCheck; pageNo++) {
          const page = await pdf.getPage(pageNo);
          const nativeText = await nativeTextForPage(page);
          if (nativeText.length >= MIN_NATIVE_TEXT) {
            page.cleanup();
            continue;
          }

          if (!tesseractReady) {
            setStatus('Carregando motor OCR em português…', 'working');
            await loadClassicScript(`https://cdn.jsdelivr.net/npm/tesseract.js@${TESSERACT_VERSION}/dist/tesseract.min.js`);
            if (!window.Tesseract || typeof window.Tesseract.createWorker !== 'function') {
              throw new Error('Motor OCR não carregou corretamente');
            }
            worker = await window.Tesseract.createWorker('por', 1, {
              logger: (m) => {
                if (!m || typeof m.progress !== 'number') return;
                const pct = Math.round(m.progress * 100);
                if (m.status === 'recognizing text') {
                  setStatus(`OCR ${file.name} · página ${pageNo}/${pagesToCheck} · ${pct}%`, 'working');
                }
              }
            });
            tesseractReady = true;
          }

          setStatus(`Lendo ${file.name} · página ${pageNo}/${pagesToCheck}…`, 'working');
          const canvas = await renderPage(page);
          const ret = await worker.recognize(canvas);
          const text = ret && ret.data && ret.data.text ? ret.data.text.trim() : '';
          if (text.length >= MIN_OCR_TEXT) {
            pieces.push(`--- Página ${pageNo} ---\n${text}`);
            ocrPages++;
          }
          canvas.width = 1;
          canvas.height = 1;
          page.cleanup();
        }

        if (pdf.numPages > MAX_PDF_PAGES) {
          pieces.push(`--- Observação ---\nOCR automático limitado às primeiras ${MAX_PDF_PAGES} páginas deste arquivo.`);
        }
        if (pieces.length && ocrPages > 0) {
          results.push({ name: file.name, text: pieces.join('\n\n'), pages: ocrPages });
        }
        await pdf.destroy();
      }
    } finally {
      if (worker) {
        try { await worker.terminate(); } catch (_) { /* noop */ }
      }
    }
    return results;
  };

  form.addEventListener('submit', async (event) => {
    if (form.dataset.ocrDone === '1') return;
    const files = Array.from(fileInput.files || []);
    if (!files.some((file) => file.name && file.name.toLowerCase().endsWith('.pdf'))) return;

    event.preventDefault();
    submitButton.disabled = true;
    const originalText = submitButton.textContent;
    submitButton.textContent = 'Lendo PDFs…';

    try {
      const ocr = await recognizeScannedPDFs(files);
      payloadInput.value = JSON.stringify(ocr);
      if (ocr.length) {
        const pages = ocr.reduce((sum, item) => sum + (item.pages || 0), 0);
        setStatus(`OCR concluído: ${ocr.length} arquivo(s), ${pages} página(s) recuperada(s). Enviando para análise…`, 'ok');
      } else {
        setStatus('Os PDFs já tinham texto pesquisável ou não precisaram de OCR. Enviando para análise…', 'ok');
      }
    } catch (err) {
      console.error('AutoProjeto OCR:', err);
      payloadInput.value = '[]';
      setStatus('OCR local não pôde ser concluído. O sistema continuará a análise e preservará os PDFs para conferência.', 'warning');
    } finally {
      form.dataset.ocrDone = '1';
      submitButton.disabled = false;
      submitButton.textContent = originalText;
      setTimeout(() => form.requestSubmit(submitButton), 60);
    }
  });
})();
