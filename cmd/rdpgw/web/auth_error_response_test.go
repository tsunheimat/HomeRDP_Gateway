package web

import (
	"bytes"
	"log"
	"os"
	"testing"
)

func captureTestLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	var logs bytes.Buffer
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	log.SetOutput(&logs)
	log.SetFlags(0)
	t.Cleanup(func() {
		if originalOutput == nil {
			originalOutput = os.Stderr
		}
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
	})

	return &logs
}
