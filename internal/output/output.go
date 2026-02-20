package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/ripkitten-co/filehound/internal/source"
)

type File = source.File

type Formatter interface {
	Start() error
	Write(f File) error
	End() error
}

type BaseFormatter struct {
	writer   io.Writer
	noHeader bool
}

type TableFormatter struct {
	BaseFormatter
	tw *tabwriter.Writer
}

func NewTableFormatter(w io.Writer, noHeader bool) *TableFormatter {
	return &TableFormatter{
		BaseFormatter: BaseFormatter{writer: w, noHeader: noHeader},
	}
}

func (t *TableFormatter) Start() error {
	t.tw = tabwriter.NewWriter(t.writer, 0, 0, 2, ' ', 0)
	if !t.noHeader {
		fmt.Fprintln(t.tw, "PATH\tSIZE\tMODIFIED")
	}
	return nil
}

func (t *TableFormatter) Write(f File) error {
	size := formatSize(f.Size)
	modTime := formatTime(f.ModTime)
	_, err := fmt.Fprintf(t.tw, "%s\t%s\t%s\n", f.Path, size, modTime)
	return err
}

func (t *TableFormatter) End() error {
	return t.tw.Flush()
}

type JSONFormatter struct {
	BaseFormatter
	encoder *json.Encoder
}

func NewJSONFormatter(w io.Writer, noHeader bool) *JSONFormatter {
	return &JSONFormatter{
		BaseFormatter: BaseFormatter{writer: w, noHeader: noHeader},
	}
}

func (j *JSONFormatter) Start() error {
	j.encoder = json.NewEncoder(j.writer)
	return nil
}

func (j *JSONFormatter) Write(f File) error {
	return j.encoder.Encode(f)
}

func (j *JSONFormatter) End() error {
	return nil
}

type CSVFormatter struct {
	BaseFormatter
	writer  *csv.Writer
	started bool
}

func NewCSVFormatter(w io.Writer, noHeader bool) *CSVFormatter {
	return &CSVFormatter{
		BaseFormatter: BaseFormatter{writer: w, noHeader: noHeader},
	}
}

func (c *CSVFormatter) Start() error {
	c.writer = csv.NewWriter(c.BaseFormatter.writer)
	c.started = false
	return nil
}

func (c *CSVFormatter) Write(f File) error {
	if !c.started && !c.noHeader {
		if err := c.writer.Write([]string{"path", "size", "modtime", "mode", "is_symlink"}); err != nil {
			return err
		}
		c.started = true
	}

	record := []string{
		f.Path,
		fmt.Sprintf("%d", f.Size),
		fmt.Sprintf("%d", f.ModTime),
		f.Mode.String(),
		fmt.Sprintf("%t", f.IsSymlink),
	}
	return c.writer.Write(record)
}

func (c *CSVFormatter) End() error {
	c.writer.Flush()
	return c.writer.Error()
}

func NewFormatter(format, outFile string, noHeader bool) Formatter {
	var w io.Writer = os.Stdout

	if outFile != "" {
		f, err := os.Create(outFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create output file: %v\n", err)
			os.Exit(1)
		}
		w = f
	}

	switch format {
	case "json":
		return NewJSONFormatter(w, noHeader)
	case "csv":
		return NewCSVFormatter(w, noHeader)
	default:
		return NewTableFormatter(w, noHeader)
	}
}

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2fTB", float64(size)/TB)
	case size >= GB:
		return fmt.Sprintf("%.2fGB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2fMB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2fKB", float64(size)/KB)
	default:
		return fmt.Sprintf("%dB", size)
	}
}

func formatTime(unix int64) string {
	if unix == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", unix)
}
