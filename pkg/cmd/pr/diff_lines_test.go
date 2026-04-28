package pr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAddedLinesByFile(t *testing.T) {
	diff := `diff --git a/foo.py b/foo.py
index 0000000..1111111 100644
--- a/foo.py
+++ b/foo.py
@@ -10,3 +10,4 @@ def x():
 ctx
+added10
 ctx2
+added12
diff --git a/bar.go b/bar.go
index 2222222..3333333 100644
--- a/bar.go
+++ b/bar.go
@@ -5,2 +5,3 @@
 ctx
+goline
`
	got := ParseAddedLinesByFile(diff)
	assert.True(t, got["foo.py"][11], "foo.py:11 should be added")
	assert.True(t, got["foo.py"][13], "foo.py:13 should be added")
	assert.False(t, got["foo.py"][10], "foo.py:10 is context, not added")
	assert.True(t, got["bar.go"][6], "bar.go:6 should be added")
}

func TestParseAddedLinesByFile_Empty(t *testing.T) {
	assert.Equal(t, map[string]map[int]bool{}, ParseAddedLinesByFile(""))
}

func TestParseAddedLinesByFile_RemovedLinesIgnored(t *testing.T) {
	diff := `diff --git a/x.go b/x.go
--- a/x.go
+++ b/x.go
@@ -1,4 +1,3 @@
 keep
-gone
 keep2
 keep3
`
	got := ParseAddedLinesByFile(diff)
	assert.Empty(t, got["x.go"], "removed-only hunk has no added lines")
}
