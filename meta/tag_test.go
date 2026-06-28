// Created at 2026-06-28
package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTag_Empty(t *testing.T) {
	opts := ParseTag("")
	assert.Equal(t, TagOptions{}, opts)
}

func TestParseTag_Col(t *testing.T) {
	opts := ParseTag("col=user_name")
	assert.Equal(t, "user_name", opts.Col)
}

func TestParseTag_PK(t *testing.T) {
	opts := ParseTag("pk")
	assert.True(t, opts.IsPK)
}

func TestParseTag_Ref(t *testing.T) {
	opts := ParseTag("ref=users.id")
	assert.Equal(t, "users", opts.RefTable)
	assert.Equal(t, "id", opts.RefCol)
}

func TestParseTag_Ref_NoColumn(t *testing.T) {
	opts := ParseTag("ref=users")
	assert.Equal(t, "users", opts.RefTable)
	assert.Equal(t, "", opts.RefCol)
}

func TestParseTag_Prefix(t *testing.T) {
	opts := ParseTag("prefix=audit_")
	assert.Equal(t, "audit_", opts.Prefix)
}

func TestParseTag_Readonly(t *testing.T) {
	opts := ParseTag("readonly")
	assert.True(t, opts.Readonly)
}

func TestParseTag_Auto(t *testing.T) {
	opts := ParseTag("auto")
	assert.True(t, opts.Auto)
}

func TestParseTag_Omit(t *testing.T) {
	opts := ParseTag("omit")
	assert.True(t, opts.Omit)
}

func TestParseTag_Combined(t *testing.T) {
	opts := ParseTag("col=name;pk;ref=users.id;prefix=audit_;readonly;auto;omit")
	assert.Equal(t, "name", opts.Col)
	assert.True(t, opts.IsPK)
	assert.Equal(t, "users", opts.RefTable)
	assert.Equal(t, "id", opts.RefCol)
	assert.Equal(t, "audit_", opts.Prefix)
	assert.True(t, opts.Readonly)
	assert.True(t, opts.Auto)
	assert.True(t, opts.Omit)
}

func TestParseTag_Spaces(t *testing.T) {
	opts := ParseTag(" col=name ; pk ")
	assert.Equal(t, "name", opts.Col)
	assert.True(t, opts.IsPK)
}
