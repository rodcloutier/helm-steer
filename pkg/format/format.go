package format

import (
	"io"
	"os"
	"runtime"

	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
)

// GetColorWriter returns a writer that is color capable.
func GetColorWriter(w io.Writer) io.Writer {
	if runtime.GOOS == "windows" {
		if fw, ok := w.(*os.File); ok {
			w = colorable.NewColorable(fw)
		}
	}

	return w
}

var (
	// The Red color.
	Red = color.New(color.FgHiRed)
	// The Cyan color.
	Cyan = color.New(color.FgCyan)
	// The White color.
	White = color.New(color.FgHiWhite)
	// The Yellow color.
	Yellow = color.New(color.FgHiYellow)
)

type colorizedWriter struct {
	io.Writer
	Color *color.Color
}

func (w colorizedWriter) Write(b []byte) (int, error) {
	return w.Color.Fprint(w.Writer, string(b))
}

// ColorizeWriter returns a writer which writes in the specified color.
func ColorizeWriter(w io.Writer, color *color.Color) io.Writer {
	return &colorizedWriter{Writer: GetColorWriter(w), Color: color}
}

// Print an error to the specified writer.
func Ferror(w io.Writer, err error) {
	w = GetColorWriter(w)
	Red.Fprintf(w, "%s\n", err)
}

// Highlight a given string.
func Highlight(s string) string {
	return Yellow.Sprintf("%s", s)
}

// Important marks a given string as being important.
func Important(s string) string {
	return White.Sprintf("%s", s)
}

func Error(s string) string {
	return Red.Sprintf("%s", s)
}
