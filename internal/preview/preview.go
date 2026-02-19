package preview

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

type Previewer struct {
	pager     string
	lineLimit int
	noColor   bool
}

type PreviewOption func(*Previewer)

func WithPager(pager string) PreviewOption {
	return func(p *Previewer) {
		if pager != "" {
			p.pager = pager
		}
	}
}

func WithLineLimit(limit int) PreviewOption {
	return func(p *Previewer) {
		if limit > 0 {
			p.lineLimit = limit
		}
	}
}

func WithNoColor(noColor bool) PreviewOption {
	return func(p *Previewer) {
		p.noColor = noColor
	}
}

func NewPreviewer(opts ...PreviewOption) *Previewer {
	p := &Previewer{
		lineLimit: 50,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.pager == "" {
		p.pager = detectPager()
	}

	return p
}

func detectPager() string {
	for _, pager := range []string{"bat", "less", "more"} {
		if _, err := exec.LookPath(pager); err == nil {
			return pager
		}
	}
	return ""
}

func (p *Previewer) Preview(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if p.lineLimit > 0 {
		data = limitLines(data, p.lineLimit)
	}

	if p.pager == "" {
		return p.printToStdout(data)
	}

	return p.pipeToPager(data)
}

func (p *Previewer) printToStdout(data []byte) error {
	_, err := os.Stdout.Write(data)
	return err
}

func (p *Previewer) pipeToPager(data []byte) error {
	var cmd *exec.Cmd

	switch p.pager {
	case "bat":
		args := []string{"--style=plain"}
		if p.lineLimit > 0 {
			args = append(args, "--line-range", fmt.Sprintf(":%d", p.lineLimit))
		}
		if p.noColor {
			args = append(args, "--decorations=never")
		}
		cmd = exec.Command(p.pager, args...)
	case "less":
		args := []string{"-R", "-F"}
		cmd = exec.Command(p.pager, args...)
	case "more":
		cmd = exec.Command(p.pager)
	default:
		cmd = exec.Command(p.pager)
	}

	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func limitLines(data []byte, limit int) []byte {
	lines := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			lines++
			if lines >= limit {
				return data[:i+1]
			}
		}
	}
	return data
}

func Preview(path string, lineLimit int) error {
	p := NewPreviewer(WithLineLimit(lineLimit))
	return p.Preview(path)
}

func HasPager() bool {
	return detectPager() != ""
}

func IsTerminal(fd int) bool {
	if runtime.GOOS == "windows" {
		return isTerminalWindows(fd)
	}
	return isTerminalUnix(fd)
}

func isTerminalWindows(fd int) bool {
	handle := os.NewFile(uintptr(fd), "")
	if handle == nil {
		return false
	}
	defer handle.Close()

	_, err := exec.Command("mode", "con").CombinedOutput()
	return err == nil
}

func isTerminalUnix(fd int) bool {
	file := os.NewFile(uintptr(fd), "")
	if file == nil {
		return false
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return false
	}

	return (fi.Mode() & os.ModeCharDevice) != 0
}

func Copy(dst io.Writer, src io.Reader, limit int64) (written int64, err error) {
	if limit <= 0 {
		return io.Copy(dst, src)
	}
	return io.CopyN(dst, src, limit)
}
