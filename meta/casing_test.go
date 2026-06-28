package meta_test

import (
	"testing"

	"github.com/mirrorru/qqm/meta"
	"github.com/stretchr/testify/assert"
)

func TestToSnakeCase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{in: "test", want: "test"},
		{in: "Test", want: "test"},
		{in: "TestOne", want: "test_one"},
		{in: "testOne", want: "test_one"},
		{in: "User5", want: "user5"},
	}
	for _, tt := range tests {
		got := meta.ToSnakeCase(tt.in)
		assert.Equal(t, tt.want, got)
	}
}
