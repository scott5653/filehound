package cmd

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ripkitten-co/filehound/internal/matcher"
	"github.com/ripkitten-co/filehound/internal/scanner"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename [PATH...]",
	Short: "Batch rename files using patterns",
	Long: `Batch rename files using template patterns.

Template variables:
  {{name}}    - original filename without extension
  {{ext}}     - file extension (including dot)
  {{size}}    - file size in bytes
  {{sha1:N}}  - first N characters of SHA1 hash (default: 8)
  {{sha256:N}} - first N characters of SHA256 hash (default: 8)

Examples:
  filehound rename . --glob "*.jpg" --pattern "img_{{sha1:8}}{{ext}}" --dry-run
  filehound rename ./photos --pattern "{{size}}_{{name}}{{ext}}"`,
	Args: cobra.MinimumNArgs(0),
	Run:  runRename,
}

func init() {
	rootCmd.AddCommand(renameCmd)

	renameCmd.Flags().StringP("pattern", "p", "", "rename pattern template (required)")
	renameCmd.Flags().Bool("dry-run", false, "preview changes without applying")
	renameCmd.Flags().StringP("glob", "g", "", "glob pattern for filename")
	renameCmd.Flags().StringSlice("ext", []string{}, "file extensions to match")
	renameCmd.Flags().String("size", "", "size filter")
	renameCmd.Flags().IntP("workers", "w", 0, "number of parallel workers")
	renameCmd.Flags().Bool("force", false, "overwrite existing files")

	_ = renameCmd.MarkFlagRequired("pattern")
}

func runRename(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	pattern, _ := cmd.Flags().GetString("pattern")
	if pattern == "" {
		fmt.Fprintln(os.Stderr, "pattern is required")
		os.Exit(1)
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	workers, _ := cmd.Flags().GetInt("workers")
	if workers <= 0 {
		workers = 8
	}

	s := scanner.New(
		scanner.WithPaths(paths...),
		scanner.WithWorkers(workers),
	)

	results, err := s.Scan()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	matchers := buildRenameMatchers(cmd)

	planner := NewRenamePlanner(pattern, dryRun, force)

	for {
		select {
		case <-ctx.Done():
			return
		case r, ok := <-results:
			if !ok {
				planner.PrintSummary()
				return
			}
			if r.Err != nil {
				continue
			}

			if len(matchers) > 0 {
				matched := true
				for _, m := range matchers {
					if !m.Match(r.File) {
						matched = false
						break
					}
				}
				if !matched {
					continue
				}
			}

			if err := planner.Plan(r.File); err != nil {
				fmt.Fprintf(os.Stderr, "error planning rename for %s: %v\n", r.File.Path, err)
			}
		}
	}
}

func buildRenameMatchers(cmd *cobra.Command) []matcher.Matcher {
	var matchers []matcher.Matcher

	glob, _ := cmd.Flags().GetString("glob")
	if glob != "" {
		matchers = append(matchers, matcher.NewGlobMatcher(glob))
	}

	exts, _ := cmd.Flags().GetStringSlice("ext")
	if len(exts) > 0 {
		matchers = append(matchers, matcher.NewExtensionMatcher(exts))
	}

	sizeFilter, _ := cmd.Flags().GetString("size")
	if sizeFilter != "" {
		size, err := matcher.ParseSize(sizeFilter[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid size: %v\n", err)
			os.Exit(1)
		}
		matchers = append(matchers, matcher.NewSizeMatcher(sizeFilter[:1], size))
	}

	return matchers
}

type RenamePlan struct {
	OldPath string
	NewPath string
}

type RenamePlanner struct {
	pattern string
	dryRun  bool
	force   bool
	plans   []RenamePlan
}

func NewRenamePlanner(pattern string, dryRun, force bool) *RenamePlanner {
	return &RenamePlanner{
		pattern: pattern,
		dryRun:  dryRun,
		force:   force,
		plans:   make([]RenamePlan, 0),
	}
}

func (p *RenamePlanner) Plan(f scanner.File) error {
	newName, err := p.applyPattern(f)
	if err != nil {
		return err
	}

	dir := filepath.Dir(f.Path)
	newPath := filepath.Join(dir, newName)

	if newPath == f.Path {
		return nil
	}

	plan := RenamePlan{
		OldPath: f.Path,
		NewPath: newPath,
	}

	if !p.dryRun {
		if _, err := os.Stat(newPath); err == nil && !p.force {
			fmt.Fprintf(os.Stderr, "skipping %s: destination exists %s (use --force to overwrite)\n", f.Path, newPath)
			return nil
		}

		if err := os.Rename(f.Path, newPath); err != nil {
			return fmt.Errorf("rename failed: %w", err)
		}
		fmt.Printf("renamed: %s -> %s\n", f.Path, newPath)
	} else {
		fmt.Printf("[dry-run] %s -> %s\n", f.Path, newPath)
	}

	p.plans = append(p.plans, plan)
	return nil
}

func (p *RenamePlanner) PrintSummary() {
	if len(p.plans) == 0 {
		fmt.Println("no files matched")
		return
	}

	if p.dryRun {
		fmt.Printf("\n[dry-run] %d files would be renamed\n", len(p.plans))
	} else {
		fmt.Printf("\n%d files renamed\n", len(p.plans))
	}
}

func (p *RenamePlanner) applyPattern(f scanner.File) (string, error) {
	result := p.pattern

	result = strings.ReplaceAll(result, "{{name}}", fileNameWithoutExt(f.Path))
	result = strings.ReplaceAll(result, "{{ext}}", filepath.Ext(f.Path))
	result = strings.ReplaceAll(result, "{{size}}", fmt.Sprintf("%d", f.Size))

	result = p.applyHashTemplates(result, f.Path)

	return result, nil
}

func (p *RenamePlanner) applyHashTemplates(result, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return result
	}

	for {
		sha1Idx := strings.Index(result, "{{sha1:")
		if sha1Idx != -1 {
			endIdx := strings.Index(result[sha1Idx:], "}}")
			if endIdx != -1 {
				nStr := result[sha1Idx+7 : sha1Idx+endIdx]
				n := 8
				_, _ = fmt.Sscanf(nStr, "%d", &n)
				if n > 40 {
					n = 40
				}

				hash := sha1.Sum(data)
				hashStr := hex.EncodeToString(hash[:])[:n]
				result = result[:sha1Idx] + hashStr + result[sha1Idx+endIdx+2:]
			} else {
				break
			}
		} else {
			break
		}
	}

	for {
		sha256Idx := strings.Index(result, "{{sha256:")
		if sha256Idx != -1 {
			endIdx := strings.Index(result[sha256Idx:], "}}")
			if endIdx != -1 {
				nStr := result[sha256Idx+9 : sha256Idx+endIdx]
				n := 8
				_, _ = fmt.Sscanf(nStr, "%d", &n)
				if n > 64 {
					n = 64
				}

				hash := sha256.Sum256(data)
				hashStr := hex.EncodeToString(hash[:])[:n]
				result = result[:sha256Idx] + hashStr + result[sha256Idx+endIdx+2:]
			} else {
				break
			}
		} else {
			break
		}
	}

	return result
}

func fileNameWithoutExt(path string) string {
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	return name[:len(name)-len(ext)]
}
