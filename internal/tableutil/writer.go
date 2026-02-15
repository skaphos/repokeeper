package tableutil

import (
	"fmt"
	"io"

	"github.com/liggitt/tabwriter"
)

// New creates a tabwriter with RepoKeeper's default spacing settings.
func New(out io.Writer, stripEscape bool) *tabwriter.Writer {
	var flags uint
	if stripEscape {
		flags = tabwriter.StripEscape
	}
	return tabwriter.NewWriter(out, 0, 4, 2, ' ', flags)
}

// PrintHeaders writes a tab-separated header row unless disabled.
func PrintHeaders(w io.Writer, noHeaders bool, headers string) error {
	if noHeaders {
		return nil
	}
	_, err := fmt.Fprintln(w, headers)
	return err
}
