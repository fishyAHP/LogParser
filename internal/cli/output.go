package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
)

type OutputMode string

const (
	OutputJSON  OutputMode = "json"
	OutputTable OutputMode = "table"

	maxTableRows = 100
	maxCellWidth = 48
)

type TablePrinter struct {
	rows         []map[string]string
	columns      []string
	seen         map[string]struct{}
	fixedColumns bool
}

func NewTablePrinter(columns ...string) *TablePrinter {
	t := &TablePrinter{
		rows: make([]map[string]string, 0, maxTableRows),
		seen: make(map[string]struct{}),
	}

	for _, name := range columns {
		if _, exists := t.seen[name]; exists {
			continue
		}

		t.seen[name] = struct{}{}
		t.columns = append(t.columns, name)
	}

	t.fixedColumns = len(columns) > 0

	return t
}

func (t *TablePrinter) AddRow(data []byte) error {
	if len(t.rows) >= maxTableRows {
		return fmt.Errorf(
			"table row limit exceeded: %d",
			maxTableRows,
		)
	}

	var raw map[string]json.RawMessage

	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("decode result: %w", err)
	}

	if raw == nil {
		return fmt.Errorf("expected JSON object")
	}

	row := make(map[string]string, len(raw))

	for name, value := range raw {
		cell, err := formatCell(value)
		if err != nil {
			return fmt.Errorf(
				"format field %q: %w",
				name,
				err,
			)
		}

		row[name] = cell

		if !t.fixedColumns {
			if _, exists := t.seen[name]; !exists {
				t.seen[name] = struct{}{}
				t.columns = append(t.columns, name)
			}
		}
	}

	t.rows = append(t.rows, row)
	return nil
}

func (t *TablePrinter) Print(out io.Writer) error {
	if len(t.rows) == 0 {
		return nil
	}

	if !t.fixedColumns {
		sort.Strings(t.columns)
	}

	columns := append(
		[]string{"№"},
		t.columns...,
	)

	widths := make([]int, len(columns))

	for i, name := range columns {
		widths[i] = min(
			maxCellWidth,
			runewidth.StringWidth(name),
		)
	}

	widths[0] = max(
		1,
		len(strconv.Itoa(len(t.rows))),
	)

	for _, row := range t.rows {
		for j, name := range t.columns {
			width := runewidth.StringWidth(
				row[name],
			)

			widths[j+1] = min(
				maxCellWidth,
				max(widths[j+1], width),
			)
		}
	}

	if err := printBorder(
		out, widths, "┌", "┬", "┐",
	); err != nil {
		return err
	}

	if err := printCells(
		out, columns, widths,
	); err != nil {
		return err
	}

	if err := printBorder(
		out, widths, "├", "┼", "┤",
	); err != nil {
		return err
	}

	for i, row := range t.rows {
		cells := make([]string, len(columns))
		cells[0] = strconv.Itoa(i + 1)

		for j, name := range t.columns {
			value, exists := row[name]
			if !exists {
				value = "NULL"
			}
			cells[j+1] = value
		}

		if err := printCells(
			out, cells, widths,
		); err != nil {
			return err
		}
	}

	return printBorder(
		out, widths, "└", "┴", "┘",
	)
}

func printBorder(
	out io.Writer,
	widths []int,
	left, middle, right string,
) error {
	var b strings.Builder

	b.WriteString(left)

	for i, width := range widths {
		b.WriteString(
			strings.Repeat("─", width+2),
		)

		if i < len(widths)-1 {
			b.WriteString(middle)
		}
	}

	b.WriteString(right)

	_, err := fmt.Fprintln(out, b.String())
	return err
}

func printCells(
	out io.Writer,
	cells []string,
	widths []int,
) error {
	var b strings.Builder

	b.WriteString("│")

	for i, cell := range cells {
		cell = truncateCell(cell, widths[i])

		b.WriteString(" ")
		b.WriteString(
			runewidth.FillRight(cell, widths[i]),
		)
		b.WriteString(" │")
	}

	_, err := fmt.Fprintln(out, b.String())
	return err
}

func truncateCell(value string, width int) string {
	if runewidth.StringWidth(value) <= width {
		return value
	}

	if width <= 1 {
		return "…"
	}

	return runewidth.Truncate(
		value,
		width,
		"…",
	)
}

func formatCell(data json.RawMessage) (string, error) {
	if len(data) == 0 {
		return "NULL", nil
	}

	var value any

	if err := json.Unmarshal(data, &value); err != nil {
		return "", err
	}

	switch v := value.(type) {
	case nil:
		return "NULL", nil

	case string:
		return escapeCell(v), nil

	case bool:
		return strconv.FormatBool(v), nil

	default:
		return escapeCell(string(data)), nil
	}
}

func escapeCell(value string) string {
	replacer := strings.NewReplacer(
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	)

	return replacer.Replace(value)
}
