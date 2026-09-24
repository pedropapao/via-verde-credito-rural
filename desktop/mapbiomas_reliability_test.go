package main

import (
	"os"
	"strings"
	"testing"
)

func TestMapBiomasUnavailableErrorIsUserFriendly193(t *testing.T) {
	err := mapBiomasUnavailableError(
		assertErr193(`Post "https://plataforma.alerta.mapbiomas.org/api/v2/graphql": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`),
	)
	if err == nil {
		t.Fatal("esperava erro amigável de indisponibilidade")
	}
	got := err.Error()
	if got != mapBiomasUnavailableMessage {
		t.Fatalf("mensagem inesperada: %q", got)
	}
	for _, technical := range []string{"https://", "context deadline exceeded", "client.timeout", "graphql"} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(technical)) {
			t.Fatalf("mensagem do usuário expôs detalhe técnico %q: %s", technical, got)
		}
	}
}

func TestEnvironmentalCacheRejectsUnavailableMapBiomas193(t *testing.T) {
	out := EnvironmentalIntelligenceResult{
		Profile: EnvironmentalProfile{
			BiomeAvailable: true,
			Biome: "Cerrado",
			Fire: EnvironmentalFireProfile{
				Status: fireStatusNone,
				Checked: true,
				Available: true,
			},
			Hydrology: EnvironmentalHydrologyProfile{
				Status: hydrologyStatusNone,
				Checked: true,
				Available: true,
			},
		},
		MapBiomas: MapBiomasCARSummary{
			Connected: true,
			Available: false,
			Message: mapBiomasUnavailableMessage,
		},
	}
	if environmentalIntelligenceCacheUsable(out) {
		t.Fatal("cache com MapBiomas indisponível não deve ser reutilizado")
	}

	out.MapBiomas.Available = true
	if !environmentalIntelligenceCacheUsable(out) {
		t.Fatal("cache com MapBiomas consultado deveria ser reutilizável")
	}
}

func TestMapBiomasUnavailableUIIsNotZero193(t *testing.T) {
	for _, path := range []string{
		"frontend/dist/ui150.js",
		"frontend/dist/ui180.js",
	} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		js := string(b)
		if !strings.Contains(js, "mb.available") {
			t.Fatalf("%s não verifica disponibilidade do MapBiomas", path)
		}
		if !strings.Contains(js, "Indisponível") {
			t.Fatalf("%s não apresenta estado indisponível", path)
		}
	}
}

type err193 string

func (e err193) Error() string { return string(e) }

func assertErr193(s string) error { return err193(s) }


func TestMapBiomasReportUnavailableIsNotZero193(t *testing.T) {
	status, detail := mapBiomasReportState(MapBiomasCARSummary{
		Connected: true,
		Available: false,
		Message: mapBiomasUnavailableMessage,
	})
	if status != "Base indisponível" {
		t.Fatalf("status inesperado: %q", status)
	}
	if strings.Contains(detail, "0 alerta") {
		t.Fatalf("indisponibilidade não pode ser apresentada como zero alertas: %q", detail)
	}

	conclusion := environmentalConclusionText(EnvironmentalIntelligenceResult{
		MapBiomas: MapBiomasCARSummary{
			Connected: true,
			Available: false,
			Message: mapBiomasUnavailableMessage,
		},
	}, nil)
	if !strings.Contains(conclusion, "indisponível") ||
		!strings.Contains(conclusion, "não significa ausência de alertas") {
		t.Fatalf("conclusão deveria registrar indisponibilidade: %q", conclusion)
	}
}
