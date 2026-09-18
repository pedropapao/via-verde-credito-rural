package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestScanDBFForCAR(t *testing.T) {
	fields := []struct {
		name string
		size byte
	}{
		{"COD_IMOVEL", 60},
		{"STATUS", 20},
	}
	headerLen := uint16(32 + len(fields)*32 + 1)
	recordLen := uint16(1 + 60 + 20)

	var b bytes.Buffer
	header := make([]byte, 32)
	header[0] = 0x03
	binary.LittleEndian.PutUint32(header[4:8], 1)
	binary.LittleEndian.PutUint16(header[8:10], headerLen)
	binary.LittleEndian.PutUint16(header[10:12], recordLen)
	b.Write(header)
	for _, f := range fields {
		desc := make([]byte, 32)
		copy(desc[:11], []byte(f.name))
		desc[11] = 'C'
		desc[16] = f.size
		b.Write(desc)
	}
	b.WriteByte(0x0D)

	car := "MG-3107703-7186.E27C.4DE7.4C13.A051.ACE1.4DAF.E97C"
	record := bytes.Repeat([]byte{' '}, int(recordLen))
	record[0] = ' '
	copy(record[1:1+60], []byte(car))
	copy(record[1+60:], []byte("ATIVO"))
	b.Write(record)

	found, got, err := scanDBFForCAR(bytes.NewReader(b.Bytes()), car, "MG-3107703-7186E27C4DE74C13A051ACE14DAFE97C")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("CAR não localizado no DBF sintético")
	}
	if got["COD_IMOVEL"] != car {
		t.Fatalf("CAR lido incorretamente: %q", got["COD_IMOVEL"])
	}
	if got["STATUS"] != "ATIVO" {
		t.Fatalf("status lido incorretamente: %q", got["STATUS"])
	}
}
