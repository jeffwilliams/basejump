package diff

import (
	"fmt"
	"strconv"
	"unicode"
)

type TokType int

const (
	TypeContextLine TokType = iota
	TypeAddedLine
	TypeRemovedLine
	// Range Line is the first hunk starting line in a context
	// or unified diff, like @@ -567,6 +572,18 @@ ...
	TypeRange
	TypeOrigFile
	TypeModifiedFile
	// TypeDiffLine is currently not parsed and returned.
	TypeDiffLine
	TypeIndexLine
	TypeOther
)

type LineTok string
type RangeTok struct {
	// OrigStartLine is the line in the original file where
	// this hunk starts.
	OrigStartLine int
	// OrigHunkLines is the number of lines in the original file
	// that the hunk covers
	OrigHunkLines int
	// ModStartLine is the line in the original file where
	// this hunk starts.
	ModStartLine int
	// ModHunkLines is the number of lines in the original file
	// that the hunk covers
	ModHunkLines int
}

type FileTok string
type Other string

func ParseDiffLine(l string) (typ TokType, tok interface{}) {

	// Every line in the diff must be at least one character
	if len(l) < 1 {
		typ = TypeOther
		tok = Other(l)
		return
	}

	switch l[0] {
	case '+':
		// Either added lines, or the file line i.e: +++ b/main.go
		if len(l) > 4 && l[0:4] == "+++ " {
			typ = TypeModifiedFile
			tok = parseFilePathLine(l)
		} else {
			typ = TypeAddedLine
			tok = FileTok(l)
		}
	case '-':
		if len(l) > 4 && l[0:4] == "--- " {
			typ = TypeOrigFile
			tok = parseFilePathLine(l)
		} else {
			typ = TypeRemovedLine
			tok = FileTok(l)
		}
	case ' ':
		typ = TypeContextLine
		tok = FileTok(l)
	case '@':
		typ = TypeRange
		var err error
		tok, err = parseRange(l)
		if err != nil {
			typ = TypeOther
			tok = Other(l)
		}
	default:
		typ = TypeOther
		tok = Other(l)

	}
	return
}

// parseFilePathLine parses a line of the form "+++ b/main.go" or "--- a/main.go"
func parseFilePathLine(l string) FileTok {
	file := l[4:]
	if len(file) > 2 && (file[0:2] == "a/" || file[0:2] == "b/") {
		file = file[2:]
	}
	return FileTok(file)
}

func parseRange(l string) (tok RangeTok, err error) {
	// @@ -121,12 +121,24 @@ random text

	var i int

	runes := []rune(l)

	// Scan a sequence of digits, moving i along to the first character after.
	scanDigits := func() (start, end int) {
		if err != nil {
			return
		}

		start = i
		for ; unicode.IsDigit(runes[i]); i++ {
		}
		end = i // One past the end
		return
	}

	expect := func(expected string) {
		if err != nil {
			return
		}

		for _, c := range expected {
			if runes[i] != c {
				err = fmt.Errorf("Invalid range line %s: expected '%c' at offset %d", l, c, i)
				return
			}
			i++
		}
	}

	// Scan an int and then expect a certain sequence after it
	scanIntThenExpect := func(expected string) (n int) {
		if err != nil {
			return
		}

		start, end := scanDigits()
		if end == start {
			err = fmt.Errorf("Invalid range line %s: line no. is empty at offset %d", l, i)
			return
		}

		n, err := strconv.Atoi(string(runes[start:end]))
		if err != nil {
			return
		}

		expect(expected)
		return
	}

	if len(runes) < 14 {
		err = fmt.Errorf("Invalid range line %s", l)
	}

	expect("@@ -")
	tok.OrigStartLine = scanIntThenExpect(",")
	tok.OrigHunkLines = scanIntThenExpect(" +")
	tok.ModStartLine = scanIntThenExpect(",")
	tok.ModHunkLines = scanIntThenExpect(" @@")
	return
}

type LineGetter interface {
	CurrentLineNumber() (num int, err error)
	LineText(line int) (text string, err error)
}

type PathExists func(path string) bool

func CalcFileAndLine(n LineGetter, pathExists PathExists) (path string, lineNo int, err error) {

	rtok, origFile, modFile, lineCnt, err := findDiffProperties(n)
	if err != nil {
		return
	}

	// Now try and figure out what file to open and where.
	// First, start with the modified file. If it's a git diff, the file
	// may or may not be absolute (i.e. in a git diff, a/tmp/file might refer to
	// tmp/file or /tmp/file.
	// If neither of those exist, try the original file.
	isModFile := true
	found := false
	path = modFile

	for i := 0; i < 4; i++ {
		if pathExists(path) {
			found = true
			break

		}

		switch i {
		case 0:
			path = "/" + path
		case 1:
			path = origFile
			isModFile = false
		case 2:
			path = "/" + path
		case 3:
			break
		}
	}

	if !found {
		err = fmt.Errorf("Can't locate the file to open.")
		return
	}

	if isModFile {
		lineNo = lineCnt + rtok.ModStartLine
	} else {
		lineNo = lineCnt + rtok.OrigStartLine
	}

	return
}

func findDiffProperties(n LineGetter) (rtok RangeTok, origFile, modFile string, lineCnt int, err error) {

	lineno, err := n.CurrentLineNumber()
	if err != nil {
		return
	}

	// Assume unified diff
	// First character of the line is either ' ' (unmodified line),
	// - (removed line), or + (added line), or a hunk starting line
	// (range information) starting with @@

	// Walk back until we find the range information
	lineCnt = -1

	var found int
	const (
		foundRange = 1 << iota
		foundOrigFile
		foundModFile
	)

	for lineno >= 1 {
		var line string
		line, err = n.LineText(lineno)
		if err != nil {
			return
		}

		//fmt.Printf("findDiffProperties: parse line '%s'\n", line)
		typ, tok := ParseDiffLine(line)
		switch typ {
		case TypeRange:
			rtok = tok.(RangeTok)
			found |= foundRange
			//fmt.Printf("findDiffProperties: found range\n")
			if lineCnt == -1 {
				// It's an error to start on this line
				err = fmt.Errorf("Started on a line that is diff metainfo")
				return
			}
		case TypeOrigFile:
			origFile = string(tok.(FileTok))
			found |= foundOrigFile
			if lineCnt == -1 {
				// It's an error to start on this line
				err = fmt.Errorf("Started on a line that is diff metainfo")
				return
			}
			//fmt.Printf("findDiffProperties: found orig file\n")
		case TypeModifiedFile:
			modFile = string(tok.(FileTok))
			found |= foundModFile
			if lineCnt == -1 {
				// It's an error to start on this line
				err = fmt.Errorf("Started on a line that is diff metainfo")
				return
			}
			//fmt.Printf("findDiffProperties: found mod file\n")
		default:
			if found&foundRange == 0 {
				lineCnt++
			}
		}

		if found == 7 {
			// all found
			break
		}
		lineno--
	}
	return
}
