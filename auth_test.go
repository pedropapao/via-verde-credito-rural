package main

import "testing"

func TestPasswordHashAndVerify(t *testing.T) {
	h, err := HashPassword("SenhaMuitoForte@2026")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("SenhaMuitoForte@2026", h) {
		t.Fatal("senha válida não foi aceita")
	}
	if VerifyPassword("senha-errada", h) {
		t.Fatal("senha inválida foi aceita")
	}
}

func TestMoneyBR(t *testing.T) {
	got := formatMoney(1210500.75)
	if got != "R$ 1.210.500,75" {
		t.Fatalf("formato inesperado: %s", got)
	}
}

func TestParseFloatBRAndDot(t *testing.T) {
	if got := parseFloat("1.210.500,75"); got != 1210500.75 {
		t.Fatalf("BR: %v", got)
	}
	if got := parseFloat("1210500.75"); got != 1210500.75 {
		t.Fatalf("dot: %v", got)
	}
}
