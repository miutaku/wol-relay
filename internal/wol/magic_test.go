package wol

import "testing"

func TestBuildAndParseMagicPacket(t *testing.T) {
	mac, err := ParseMAC("00:11:22:33:44:55")
	if err != nil {
		t.Fatal(err)
	}
	packet, err := BuildMagicPacket(mac)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := ParseMagicPacket(packet)
	if !ok {
		t.Fatal("packet was not parsed as magic packet")
	}
	if got.String() != mac.String() {
		t.Fatalf("got %s, want %s", got, mac)
	}
}

func TestParseMagicPacketWithPadding(t *testing.T) {
	mac, _ := ParseMAC("001122334455")
	packet, err := BuildMagicPacket(mac)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := ParseMagicPacket(append([]byte{0, 1, 2}, packet...))
	if !ok {
		t.Fatal("packet with padding was not parsed")
	}
	if got.String() != mac.String() {
		t.Fatalf("got %s, want %s", got, mac)
	}
}
