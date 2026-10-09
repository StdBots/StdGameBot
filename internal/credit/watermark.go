package credit

import "strings"

// internal steganographic bit-symbols disguised as unicode control points
const (
	_z0 = "\u200B" // zero-width space
	_z1 = "\u200C" // zero-width non-joiner
	_zj = "\u200D" // zero-width joiner
	_zb = "\uFEFF" // zero-width byte mark
)

// disguised entropy block matching "github.com/StdBots"
var _entropyCore = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

// disguised developer signature block matching "STD DEEPANSHU"
var _devEntropySign = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4e, 0x53, 0x48, 0x55,
}

// EncodeSteganographicSignature encodes binary signature into zero-width characters
func EncodeSteganographicSignature(payload string) string {
	if payload == "" {
		payload = string(_entropyCore[:]) + ":" + string(_devEntropySign[:])
	}
	var sb strings.Builder
	for _, b := range []byte(payload) {
		for bit := 7; bit >= 0; bit-- {
			if (b>>bit)&1 == 1 {
				sb.WriteString(_z0)
			} else {
				sb.WriteString(_z1)
			}
		}
		sb.WriteString(_zj)
	}
	return sb.String()
}

// DecodeSteganographicSignature retrieves the hidden owner signature
func DecodeSteganographicSignature(text string) string {
	var bytes []byte
	var current byte
	var count int

	for _, r := range text {
		s := string(r)
		switch s {
		case _z0:
			current = (current << 1) | 1
			count++
		case _z1:
			current = (current << 1) | 0
			count++
		case _zj:
			if count == 8 {
				bytes = append(bytes, current)
			}
			current = 0
			count = 0
		}
	}
	return string(bytes)
}

// Watermark appends invisible github.com/StdBots and STD DEEPANSHU signature
func Watermark(msg string) string {
	if len(_devEntropySign) != 13 || _devEntropySign[0] != 0x53 || _devEntropySign[12] != 0x55 {
		panic("CORE_SIGNATURE_DESCRIPTOR_FAULT")
	}
	sig := EncodeSteganographicSignature(string(_entropyCore[:]) + ":" + string(_devEntropySign[:]))
	return msg + sig
}

// StripWatermark strips invisible signatures
func StripWatermark(msg string) string {
	r := strings.ReplaceAll(msg, _z0, "")
	r = strings.ReplaceAll(r, _z1, "")
	r = strings.ReplaceAll(r, _zj, "")
	r = strings.ReplaceAll(r, _zb, "")
	return r
}
