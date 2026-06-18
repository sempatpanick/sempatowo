package voice

// CreateHeader creates an RTP header for voice packets.
func CreateHeader(seq uint16, ts uint32, ssrc uint32) []byte {
	header := make([]byte, 12)
	header[0] = 0x80
	header[1] = 0x78
	header[2] = byte(seq >> 8)
	header[3] = byte(seq)
	header[4] = byte(ts >> 24)
	header[5] = byte(ts >> 16)
	header[6] = byte(ts >> 8)
	header[7] = byte(ts)
	header[8] = byte(ssrc >> 24)
	header[9] = byte(ssrc >> 16)
	header[10] = byte(ssrc >> 8)
	header[11] = byte(ssrc)
	return header
}
