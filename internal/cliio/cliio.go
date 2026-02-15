package cliio

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/skaphos/repokeeper/internal/tableutil"
)

// PromptYesNo writes prompt and reads a yes/no response from input.
func PromptYesNo(out io.Writer, in io.Reader, prompt string) (bool, error) {
	if _, err := fmt.Fprint(out, prompt); err != nil {
		return false, err
	}
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	choice := strings.ToLower(strings.TrimSpace(line))
	return choice == "y" || choice == "yes", nil
}

// WriteTable renders a simple tab-separated table with optional headers.
func WriteTable(out io.Writer, stripEscape bool, noHeaders bool, headers []string, rows [][]string) error {
	w := tableutil.New(out, stripEscape)
	if !noHeaders {
		if _, err := fmt.Fprintln(w, strings.Join(headers, "\t")); err != nil {
			return err
		}
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(w, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return w.Flush()
}
