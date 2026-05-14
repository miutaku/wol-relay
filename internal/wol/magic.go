package wol

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
)

const MagicPacketLen = 102

func ParseMAC(value string) (net.HardwareAddr, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("mac address is required")
	}
	hw, err := net.ParseMAC(value)
	if err == nil {
		if len(hw) != 6 {
			return nil, fmt.Errorf("mac address must be 6 bytes: %s", value)
		}
		return hw, nil
	}
	clean := strings.NewReplacer(":", "", "-", "", ".", "").Replace(value)
	if len(clean) != 12 {
		return nil, err
	}
	b, hexErr := hex.DecodeString(clean)
	if hexErr != nil {
		return nil, err
	}
	return net.HardwareAddr(b), nil
}

func BuildMagicPacket(mac net.HardwareAddr) ([]byte, error) {
	if len(mac) != 6 {
		return nil, fmt.Errorf("mac address must be 6 bytes, got %d", len(mac))
	}
	packet := make([]byte, 0, MagicPacketLen)
	packet = append(packet, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff)
	for i := 0; i < 16; i++ {
		packet = append(packet, mac...)
	}
	return packet, nil
}

func ParseMagicPacket(packet []byte) (net.HardwareAddr, bool) {
	if len(packet) < MagicPacketLen {
		return nil, false
	}

	start := bytes.Index(packet, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	if start < 0 || len(packet[start:]) < MagicPacketLen {
		return nil, false
	}

	mac := packet[start+6 : start+12]
	for offset := start + 12; offset < start+MagicPacketLen; offset += 6 {
		if !bytes.Equal(packet[offset:offset+6], mac) {
			return nil, false
		}
	}
	return net.HardwareAddr(append([]byte(nil), mac...)), true
}

func SendMagicPacket(mac net.HardwareAddr, target string) error {
	packet, err := BuildMagicPacket(mac)
	if err != nil {
		return err
	}

	addr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	return err
}
