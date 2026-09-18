package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestParseShapefilePolygonPayload(t *testing.T) {
	var b bytes.Buffer
	_ = binary.Write(&b, binary.LittleEndian, int32(5))
	for _, v := range []float64{-46.0, -20.0, -45.99, -19.99} {
		_ = binary.Write(&b, binary.LittleEndian, v)
	}
	_ = binary.Write(&b, binary.LittleEndian, int32(1))
	_ = binary.Write(&b, binary.LittleEndian, int32(5))
	_ = binary.Write(&b, binary.LittleEndian, int32(0))
	for _, p := range [][2]float64{
		{-46.0, -20.0},
		{-45.99, -20.0},
		{-45.99, -19.99},
		{-46.0, -19.99},
		{-46.0, -20.0},
	} {
		_ = binary.Write(&b, binary.LittleEndian, p[0])
		_ = binary.Write(&b, binary.LittleEndian, p[1])
	}
	bbox, geom, err := parseShapefilePolygonPayload(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if bbox[0] != -46 || bbox[2] != -45.99 {
		t.Fatalf("bbox inesperado: %+v", bbox)
	}
	if geom.Type != "MultiPolygon" || len(geom.Coordinates) == 0 {
		t.Fatalf("geometria inesperada: %+v", geom)
	}
	if area := carGeometryAreaHa(geom); area < 100 || area > 130 {
		t.Fatalf("área inesperada %.2f ha", area)
	}
}

func TestThemeDefinitionsContainProfessionalCore(t *testing.T) {
	want := map[string]bool{"APP": false, "RESERVA_LEGAL": false, "VEGETACAO_NATIVA": false, "AREA_CONSOLIDADA": false}
	for _, d := range sicarThemeDefinitions {
		if _, ok := want[d.Code]; ok {
			want[d.Code] = true
		}
	}
	for code, ok := range want {
		if !ok {
			t.Fatalf("tema obrigatório ausente: %s", code)
		}
	}
}
