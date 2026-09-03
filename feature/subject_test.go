package feature

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubjectWith(t *testing.T) {
	t.Parallel()

	var base Subject

	next := base.With(dimUser, "u1")
	assert.Nil(t, base)
	assert.Equal(t, Subject{dimUser: "u1"}, next)

	another := next.With(dimInstallation, "i1")
	assert.Equal(t, Subject{dimUser: "u1"}, next)
	assert.Equal(t, Subject{dimUser: "u1", dimInstallation: "i1"}, another)
}

func TestSubjectMerge(t *testing.T) {
	t.Parallel()

	base := Subject{dimUser: "u1", dimInstallation: "i1"}

	merged := base.Merge(Subject{dimUser: "u2"})
	assert.Equal(t, Subject{dimUser: "u2", dimInstallation: "i1"}, merged)
	assert.Equal(t, Subject{dimUser: "u1", dimInstallation: "i1"}, base)
	assert.Equal(t, base, base.Merge(nil))
}

func TestSubjectGet(t *testing.T) {
	t.Parallel()

	subject := Subject{dimUser: "u1", dimInstallation: ""}

	value, ok := subject.Get(dimUser)
	assert.True(t, ok)
	assert.Equal(t, "u1", value)

	_, ok = subject.Get(dimInstallation)
	assert.False(t, ok, "empty values are absent")

	_, ok = subject.Get(Host)
	assert.False(t, ok)

	var nilSubject Subject

	_, ok = nilSubject.Get(dimUser)
	assert.False(t, ok)
}

func TestContextSubject(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	assert.Nil(t, SubjectFrom(ctx))

	ctx = WithAttribute(ctx, dimUser, "u1")
	assert.Equal(t, Subject{dimUser: "u1"}, SubjectFrom(ctx))

	merged := WithSubject(ctx, Subject{dimInstallation: "i1", dimUser: "u2"})
	assert.Equal(t, Subject{dimUser: "u1"}, SubjectFrom(ctx), "parent context is unchanged")
	assert.Equal(t, Subject{dimUser: "u2", dimInstallation: "i1"}, SubjectFrom(merged))
}
