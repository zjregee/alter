package usage

import (
	"bufio"
	"os"
)

type Line struct {
	Bytes        []byte
	WasDiscarded bool
}

func ScanJSONL(filePath string, maxLineBytes int, onLine func(Line)) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)

	const maxScanTokenSize = 1024 * 1024
	buf := make([]byte, 0, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		lineBytes := scanner.Bytes()

		var wasDiscarded bool
		var lineToSend []byte

		if maxLineBytes > 0 && len(lineBytes) > maxLineBytes {
			wasDiscarded = true
			lineToSend = nil
		} else {
			wasDiscarded = false
			lineToSend = make([]byte, len(lineBytes))
			copy(lineToSend, lineBytes)
		}

		onLine(Line{
			Bytes:        lineToSend,
			WasDiscarded: wasDiscarded,
		})
	}

	return scanner.Err()
}
