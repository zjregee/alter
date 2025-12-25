package usage

import (
	"bufio"
	"os"
)

type Line struct {
	Bytes        []byte
	WasTruncated bool
}

// ScanJSONL reads a .jsonl file, calling onLine for each line.
// It can truncate lines that are too long based on maxLineBytes or prefixBytes.
func ScanJSONL(fileURL string, maxLineBytes int, prefixBytes int, onLine func(Line)) error {
	file, err := os.Open(fileURL)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)

	// Set a large buffer for scanner, similar to what Swift code might be doing with its chunking.
	// 1MB should be enough for most log lines.
	const maxScanTokenSize = 1024 * 1024
	buf := make([]byte, 0, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		lineBytes := scanner.Bytes() // This is a single line from the file.

		var wasTruncated bool
		var lineToSend []byte

		// The logic from the Swift code suggests truncation if the line itself is too long.
		if maxLineBytes > 0 && len(lineBytes) > maxLineBytes {
			wasTruncated = true
			lineToSend = nil
		} else if prefixBytes > 0 && len(lineBytes) > prefixBytes {
			wasTruncated = true
			lineToSend = nil
		} else {
			wasTruncated = false
			// scanner.Bytes() is only valid until the next Scan(). We must copy it.
			lineToSend = make([]byte, len(lineBytes))
			copy(lineToSend, lineBytes)
		}

		onLine(Line{
			Bytes:        lineToSend,
			WasTruncated: wasTruncated,
		})
	}

	return scanner.Err()
}
