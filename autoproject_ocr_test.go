package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseAutoOCRPayload(t *testing.T) {
	raw := `[{"name":"CAR.pdf","text":"PRODUTOR JOSE CPF 123.456.789-00","pages":2}]`
	items := parseAutoOCRPayload(raw)
	if len(items) != 1 {
		t.Fatalf("esperava 1 item, obteve %d", len(items))
	}
	if items[0].Name != "CAR.pdf" || items[0].Pages != 2 {
		t.Fatalf("item OCR inesperado: %#v", items[0])
	}
	if !strings.Contains(items[0].Text, "PRODUTOR") {
		t.Fatalf("texto OCR não preservado: %q", items[0].Text)
	}
}

func TestParseAutoOCRPayloadRejectsInvalid(t *testing.T) {
	if got := parseAutoOCRPayload(`{quebrado`); len(got) != 0 {
		t.Fatalf("JSON inválido deveria ser ignorado, obteve %#v", got)
	}
	if got := parseAutoOCRPayload(`[{"name":"x.pdf","text":""}]`); len(got) != 0 {
		t.Fatalf("texto vazio deveria ser ignorado, obteve %#v", got)
	}
}

func TestAutoProjectOCRAssetsEmbedded(t *testing.T) {
	js, err := webFS.ReadFile("web/static/autoproject_ocr.js")
	if err != nil {
		t.Fatal(err)
	}
	text := string(js)
	for _, want := range []string{"pdfjs-dist@${PDFJS_VERSION}", "tesseract.js@${TESSERACT_VERSION}", "createWorker('por'", "ocr_payload"} {
		if !strings.Contains(text, want) {
			t.Fatalf("asset OCR não contém %q", want)
		}
	}

	html, err := webFS.ReadFile("web/templates/autoproject.html")
	if err != nil {
		t.Fatal(err)
	}
	h := string(html)
	if !strings.Contains(h, `name="ocr_payload"`) || !strings.Contains(h, `/static/autoproject_ocr.js`) {
		t.Fatal("template do AutoProjeto não ativou o OCR")
	}
}

func TestSecurityHeadersAllowOCRDependencies(t *testing.T) {
	h := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/autoproject", nil))
	csp := rr.Header().Get("Content-Security-Policy")
	for _, want := range []string{"https://cdn.jsdelivr.net", "https://tessdata.projectnaptha.com", "worker-src", "blob:", "wasm-unsafe-eval"} {
		if !strings.Contains(csp, want) {
			t.Fatalf("CSP precisa permitir %q para OCR: %s", want, csp)
		}
	}
}
