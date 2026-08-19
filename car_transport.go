package main

import (
	"crypto/tls"
	"net/http"
	"strings"
)

// sicarCompatibleTransport mantém o transporte HTTP padrão para todo o sistema
// e usa uma configuração TLS compatível somente com o GeoServer público do SICAR.
// Go 1.22+ retirou suites com troca de chaves RSA da lista padrão; o GeoServer
// público ainda pode precisar delas em alguns ambientes.
type sicarCompatibleTransport struct {
	normal *http.Transport
	legacy *http.Transport
}

func init() {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return
	}

	normal := base.Clone()
	legacy := base.Clone()
	legacy.TLSClientConfig = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		CipherSuites: sicarCipherSuites(),
	}

	http.DefaultTransport = &sicarCompatibleTransport{normal: normal, legacy: legacy}
}

func sicarCipherSuites() []uint16 {
	ids := make([]uint16, 0, 16)
	seen := map[uint16]bool{}
	for _, suite := range tls.CipherSuites() {
		if !seen[suite.ID] {
			ids = append(ids, suite.ID)
			seen[suite.ID] = true
		}
	}
	// Suites RSA ainda implementadas pelo Go, mas removidas da seleção padrão
	// em versões recentes. Mantemos apenas AES; 3DES não é reabilitado.
	for _, id := range []uint16{
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	} {
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	return ids
}

func (t *sicarCompatibleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil || !strings.EqualFold(req.URL.Hostname(), "geoserver.car.gov.br") {
		return t.normal.RoundTrip(req)
	}

	resp, err := t.legacy.RoundTrip(req)
	if err == nil && resp != nil && resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusBadGateway && resp.StatusCode != http.StatusServiceUnavailable {
		return resp, nil
	}

	// Alguns ambientes do GeoServer respondem melhor pelo endpoint /wfs do
	// workspace do que pelo endpoint genérico /ows. Fazemos a segunda tentativa
	// somente para o host oficial e somente em requisições GET sem corpo.
	if req.Method == http.MethodGet && strings.Contains(req.URL.Path, "/geoserver/sicar/ows") {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		alt := req.Clone(req.Context())
		altURL := *req.URL
		altURL.Path = strings.Replace(altURL.Path, "/geoserver/sicar/ows", "/geoserver/sicar/wfs", 1)
		alt.URL = &altURL
		return t.legacy.RoundTrip(alt)
	}

	return resp, err
}
