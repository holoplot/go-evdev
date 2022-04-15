package parser

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInputH(t *testing.T) {
	fd, err := os.Open("test_data/input.h")
	assert.NoError(t, err)

	cp := NewCodeProcessor(SelectedPrefixesGroups["input.h"])

	groups, err := cp.ProcessFile(fd)
	assert.NoError(t, err)

	data := GenerateFile(groups, false, "test", "testurl", "testurl")

	v, err := os.ReadFile("test_data/input.h-expected-output.txt")
	assert.NoError(t, err)
	assert.Equal(t, string(v), data)
}

func TestTestInputEventCodesH(t *testing.T) {
	fd, err := os.Open("test_data/input-event-codes.h")
	assert.NoError(t, err)

	cp := NewCodeProcessor(SelectedPrefixesGroups["input-event-codes.h"])

	groups, err := cp.ProcessFile(fd)
	assert.NoError(t, err)

	data := GenerateFile(groups, false, "test", "testurl", "testurl")

	v, err := os.ReadFile("test_data/input-event-codes.h-expected-output.txt")
	assert.NoError(t, err)
	assert.Equal(t, string(v), data)
}
