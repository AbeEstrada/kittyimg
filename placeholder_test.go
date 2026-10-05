/*
   Copyright 2021-2026 Olivier Mengué.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package kittyimg_test

import (
	"os"
	"strings"
	"testing"

	"github.com/dolmen-go/kittyimg"
)

func ExampleFprintPlaceholders() {
	// Image 42 has already been transmitted (a=t,i=42,q=2).
	// Create a virtual placement of 4 columns and 2 rows...
	os.Stdout.WriteString("\033_Ga=p,U=1,q=2,i=42,c=4,r=2\033\\")
	// ...and display it at the cursor.
	kittyimg.FprintPlaceholders(os.Stdout, 42, 4, 2)
}

func TestFprintPlaceholders(t *testing.T) {
	for _, tc := range []struct {
		name       string
		id         uint32
		cols, rows int
		expected   string
	}{{
		// Example from https://sw.kovidgoyal.net/kitty/graphics-protocol/#unicode-placeholders
		name: "spec", id: 42, cols: 2, rows: 2,
		expected: "\x1b[38;5;42m\U0010EEEE̅̅\U0010EEEE̅̍\x1b[39m\n" +
			"\x1b[38;5;42m\U0010EEEE̍̅\U0010EEEE̍̍\x1b[39m\n",
	}, {
		// Example from the spec, with the most significant byte of the image ID.
		name: "spec-msb", id: 42 + 2<<24, cols: 2, rows: 2,
		expected: "\x1b[38;5;42m\U0010EEEE̅̅̎\U0010EEEE̅̍̎\x1b[39m\n" +
			"\x1b[38;5;42m\U0010EEEE̍̅̎\U0010EEEE̍̍̎\x1b[39m\n",
	}, {
		name: "truecolor", id: 0x123456, cols: 1, rows: 1,
		expected: "\x1b[38;2;18;52;86m\U0010EEEE̅̅\x1b[39m\n",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder
			if err := kittyimg.FprintPlaceholders(&buf, tc.id, tc.cols, tc.rows); err != nil {
				t.Fatal(err)
			}
			if got := buf.String(); got != tc.expected {
				t.Errorf("got:\n%q\nexpected:\n%q", got, tc.expected)
			}
		})
	}
}

func TestFprintPlaceholdersErrors(t *testing.T) {
	for _, tc := range []struct {
		id         uint32
		cols, rows int
	}{
		{0, 1, 1},
		{42, 0, 1},
		{42, 1, kittyimg.MaxPlaceholderSize + 1},
	} {
		var buf strings.Builder
		if err := kittyimg.FprintPlaceholders(&buf, tc.id, tc.cols, tc.rows); err == nil {
			t.Errorf("id=%d %dx%d: error expected", tc.id, tc.cols, tc.rows)
		} else {
			t.Logf("id=%d %dx%d: %v", tc.id, tc.cols, tc.rows, err)
		}
		if buf.Len() > 0 {
			t.Errorf("id=%d %dx%d: unexpected output %q", tc.id, tc.cols, tc.rows, buf.String())
		}
	}
}

// TestPlaceholdersDiacritics checks the table of diacritics through the columns of a full row.
func TestPlaceholdersDiacritics(t *testing.T) {
	if kittyimg.MaxPlaceholderSize != 297 {
		t.Fatalf("MaxPlaceholderSize: got %d, expected 297", kittyimg.MaxPlaceholderSize)
	}

	var buf strings.Builder
	if err := kittyimg.FprintPlaceholders(&buf, 1, kittyimg.MaxPlaceholderSize, 1); err != nil {
		t.Fatal(err)
	}
	line, ok := strings.CutPrefix(buf.String(), "\x1b[38;5;1m\U0010EEEE")
	if !ok {
		t.Fatalf("unexpected prefix: %q", buf.String())
	}
	line, ok = strings.CutSuffix(line, "\x1b[39m\n")
	if !ok {
		t.Fatalf("unexpected suffix: %q", buf.String())
	}

	cells := strings.Split(line, "\U0010EEEE")
	if len(cells) != kittyimg.MaxPlaceholderSize {
		t.Fatalf("got %d cells, expected %d", len(cells), kittyimg.MaxPlaceholderSize)
	}
	var prev rune
	for col, cell := range cells {
		r := []rune(cell)
		if len(r) != 2 || r[0] != 0x0305 {
			t.Fatalf("column %d: unexpected diacritics %q", col, cell)
		}
		if r[1] <= prev {
			t.Errorf("column %d: %U is not after %U", col, r[1], prev)
		}
		prev = r[1]
	}
	if first := []rune(cells[0])[1]; first != 0x0305 {
		t.Errorf("first diacritic: got %U, expected U+0305", first)
	}
	if prev != 0x1D244 {
		t.Errorf("last diacritic: got %U, expected U+1D244", prev)
	}
}
