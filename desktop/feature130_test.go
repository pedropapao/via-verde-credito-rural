package main

import (
	"math"
	"testing"
)

func TestCalculateTerrainMetric(t *testing.T) {
	points := []GeoPoint{
		{Lat: -20.9000, Lon: -46.6000},
		{Lat: -20.9005, Lon: -46.6000},
		{Lat: -20.9010, Lon: -46.6000},
		{Lat: -20.9015, Lon: -46.6000},
	}
	elev := []float64{800, 805, 810, 812}
	m := calculateTerrainMetric(points, elev, 5)
	if !m.Available {
		t.Fatal("relevo deveria estar disponível")
	}
	if math.Abs(m.ElevationMeanM-806.75) > 0.01 {
		t.Fatalf("média inesperada: %.2f", m.ElevationMeanM)
	}
	if m.ReliefM != 12 {
		t.Fatalf("desnível inesperado: %.2f", m.ReliefM)
	}
	if m.MeanSlopePct <= 0 || m.OperationalScore <= 0 {
		t.Fatalf("métricas de relevo inválidas: %+v", m)
	}
}

func TestTerrainTilePixelBounds(t *testing.T) {
	x, y, px, py := terrainTilePixel(-20.905, -46.605, terrainTileZoom)
	if x < 0 || y < 0 || px < 0 || px > 255 || py < 0 || py > 255 {
		t.Fatalf("coordenada de tile inválida: %d/%d px=%d py=%d", x, y, px, py)
	}
}
