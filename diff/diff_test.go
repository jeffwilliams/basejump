package diff

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseRange(t *testing.T) {
	type test struct {
		name          string
		input         string
		result        RangeTok
		shouldSucceed bool
	}

	tests := []test{
		test{
			"normal",
			"@@ -121,12 +121,24 @@ func (n Basejump) CurrentWordText() (text string, err error)",
			RangeTok{121, 12, 121, 24},
			true,
		},
		test{
			"small num",
			"@@ -1,2 +2,4 @@ func (n Basejump) CurrentWordText() (text string, err error)",
			RangeTok{1, 2, 2, 4},
			true,
		},
		test{
			"missing @",
			"@ -1,2 +2,4 @@ func (n Basejump) CurrentWordText() (text string, err error)",
			RangeTok{},
			false,
		},
		test{
			"missing -",
			"@@ 1,2 +2,4 @@ func (n Basejump) CurrentWordText() (text string, err error)",
			RangeTok{},
			false,
		},
		test{
			"missing ,",
			"@@ -12 +2,4 @@ func (n Basejump) CurrentWordText() (text string, err error)",
			RangeTok{},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := parseRange(test.input)
			if test.shouldSucceed {
				if err != nil {
					t.Fatalf("Parsing returned error: %v", err)
				}

				if res != test.result {
					t.Fatalf("Result incorrect. Expected %v but got %v", test.result, res)
				}
			} else {
				if err == nil {
					t.Fatalf("Parsing didn't return error when it should have")
				}
			}
		})
	}
}

type StringLineGetter struct {
	current int
	lines   []string
}

func (s StringLineGetter) CurrentLineNumber() (num int, err error) {
	return s.current, nil
}

// Lines are numbered starting at 1
func (s StringLineGetter) LineText(line int) (text string, err error) {
	if line < 1 || line > len(s.lines) {
		return "", fmt.Errorf("Invalid line")
	}
	return s.lines[line-1], nil
}

func StringLineGetterFromString(s string, current int) StringLineGetter {
	return StringLineGetter{
		current: current,
		lines:   strings.Split(s, "\n"),
	}
}

func mkPathExists(file string) PathExists {
	return func(f string) bool {
		//fmt.Printf("PathExists called for %s\n", f)
		return f == file
	}
}

func TestCalcFileAndLine(t *testing.T) {

	diff := StringLineGetterFromString(
		`diff --git a/README.md b/README.md
index c9a2f91..54ce42b 100644
--- a/README.md
+++ b/README.md
@@ -39,6 +39,7 @@ Basejump behaves very similar to the gf, gF, CTRL-W F, etc. family of commands.
   * The visual mode form of the commands doesn't support the line number suffix of the path, only the file path itself
   * The "new window" form of the commands always opens a new window, even if one already exists for the file
   * Basejump supports http:// URLs
+  * Basejump stores the old positions within a file in the jump list, so CTRL-O and CTRL-I work
   
 
 # Installation
diff --git a/plugin/basejump.vim b/plugin/basejump.vim
index 77b49f7..498f0fe 100644
--- a/plugin/basejump.vim
+++ b/plugin/basejump.vim
@@ -22,7 +22,7 @@ function! s:RequireBasejump(host) abort
   " 'basejump' is the binary created by compiling the program.
   " If '-logpanic' is specified, panics in the binary are logged to
   " /tmp/basejump.panic
-  return jobstart([s:basejump_path], {'rpc': v:true})
+  return jobstart([s:basejump_path, '-trace'], {'rpc': v:true})
   "return jobstart([s:basejump_path,'-logpanic'], {'rpc': v:true})
 endfunction
 
@@ -46,5 +46,6 @@ nmap <M-}> :call OpenPathUnderCursor('tab')<CR>
 call remote#host#RegisterPlugin('basejump', '0', [
 \ {'type': 'function', 'name': 'OpenPathUnderCursor', 'sync': 1, 'opts': {}},
 \ {'type': 'function', 'name': 'OpenSelectedPath', 'sync': 1, 'opts': {}},
+\ {'type': 'function', 'name': 'OpenLineFromDiff', 'sync': 1, 'opts': {}},
 \ ])`,
		9)

	s, _ := diff.LineText(9)
	t.Logf("Starting line: '%s'", s)

	pathExists := mkPathExists("README.md")

	// Test at line 9
	path, lineNo, err := CalcFileAndLine(diff, pathExists)
	if err != nil {
		t.Fatalf("Got error: %v", err)
	}

	if path != "README.md" {
		t.Fatalf("Expected path to be 'README.md' but it was '%s'", path)
	}

	if lineNo != 42 {
		t.Fatalf("Expected line to be 42, but it was %d", lineNo)
	}

	// Test at line 20
	diff.current = 20
	pathExists = mkPathExists("plugin/basejump.vim")
	path, lineNo, err = CalcFileAndLine(diff, pathExists)

	if err != nil {
		t.Fatalf("Got error: %v", err)
	}

	if path != "plugin/basejump.vim" {
		t.Fatalf("Expected path to be 'plugin/basejump.vim' but it was '%s'", path)
	}

	if lineNo != 24 {
		t.Fatalf("Expected line to be 24, but it was %d", lineNo)
	}

	// Test at line 17 (start on a range line)
	diff.current = 17
	pathExists = mkPathExists("plugin/basejump.vim")
	path, lineNo, err = CalcFileAndLine(diff, pathExists)

	if err == nil {
		t.Fatalf("Expected an error but got none")
	}

	// Test at line 16 (start at 'modified file' line
	diff.current = 16
	pathExists = mkPathExists("plugin/basejump.vim")
	path, lineNo, err = CalcFileAndLine(diff, pathExists)

	if err == nil {
		t.Fatalf("Expected an error but got none")
	}

}
