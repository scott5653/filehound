package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ripkitten-co/filehound/internal/matcher"
	"github.com/ripkitten-co/filehound/internal/output"
	"github.com/ripkitten-co/filehound/internal/scanner"
	"github.com/ripkitten-co/filehound/internal/source"
	"github.com/ripkitten-co/filehound/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var scanCmd = &cobra.Command{
	Use:   "scan [PATH...]",
	Short: "Scan files matching criteria",
	Long: `Scan files in one or more directories and filter by various criteria.
Files matching all active filters are printed to stdout.`,
	Args: cobra.MinimumNArgs(0),
	Run:  runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().StringP("regex", "r", "", "regex pattern to match in file content")
	scanCmd.Flags().String("regex-path", "", "regex pattern to match in file path")
	scanCmd.Flags().Float64("entropy", 0, "minimum entropy threshold (0-8)")
	scanCmd.Flags().StringSlice("mime", []string{}, "MIME types to match (e.g., image/png,text/plain)")
	scanCmd.Flags().StringSlice("ext", []string{}, "file extensions to match (e.g., .go,.txt)")
	scanCmd.Flags().StringP("glob", "g", "", "glob pattern for filename (e.g., *.go)")
	scanCmd.Flags().String("size", "", "size filter (e.g., >1MB, <100KB, =1024)")
	scanCmd.Flags().String("modified", "", "modification time filter (e.g., <24h, >7d)")
	scanCmd.Flags().StringSlice("exclude", []string{}, "additional directories to exclude")
	scanCmd.Flags().IntP("workers", "w", 0, "number of parallel workers (default: 8)")
	scanCmd.Flags().Bool("empty", false, "match only empty files")
	scanCmd.Flags().Bool("follow", false, "follow symbolic links")
	scanCmd.Flags().BoolP("progress", "p", false, "show progress bar during scan")
	scanCmd.Flags().StringP("output", "o", "", "output format: table, json, csv (default: table)")
	scanCmd.Flags().String("out-file", "", "write output to file instead of stdout")
	scanCmd.Flags().Bool("no-header", false, "omit header row in table/CSV output")

	scanCmd.Flags().String("s3-region", "", "AWS region for S3 sources")
	scanCmd.Flags().String("s3-endpoint", "", "S3-compatible endpoint URL")
	scanCmd.Flags().String("git-mode", "working", "Git scan mode: working, full")
	scanCmd.Flags().String("git-branch", "", "Git branch to scan (for full mode)")
	scanCmd.Flags().String("git-since", "", "Scan commits since (e.g., 2024-01-01)")

	_ = viper.BindPFlags(scanCmd.Flags())
}

func runScan(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	workers, _ := cmd.Flags().GetInt("workers")
	if workers <= 0 {
		workers = 8
	}

	excludeFlags, _ := cmd.Flags().GetStringSlice("exclude")
	excludes := append(scanner.DefaultExcludes, excludeFlags...)

	follow, _ := cmd.Flags().GetBool("follow")
	showProgress, _ := cmd.Flags().GetBool("progress")

	src, err := detectSource(cmd, paths, workers, excludes, follow)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	results, err := src.List(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	matchers := buildMatchers(cmd)

	outputFormat, _ := cmd.Flags().GetString("output")
	outFile, _ := cmd.Flags().GetString("out-file")
	noHeader, _ := cmd.Flags().GetBool("no-header")

	formatter := output.NewFormatter(outputFormat, outFile, noHeader)
	if err := formatter.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = formatter.End() }()

	var progress *tui.Program
	if showProgress {
		progress = tui.NewProgressProgram()
		progress.Start()
	}

	var filesFound, errors int
	var totalBytes int64

	for {
		select {
		case <-ctx.Done():
			if progress != nil {
				progress.Quit()
			}
			return
		case r, ok := <-results:
			if !ok {
				if progress != nil {
					progress.Quit()
				}
				return
			}
			if r.Err != nil {
				errors++
				if progress != nil {
					progress.Send(tui.ProgressMsg{Errors: 1})
				}
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

			filesFound++
			totalBytes += r.File.Size

			if progress != nil {
				progress.Send(tui.ProgressMsg{
					FilesFound: 1,
					Bytes:      r.File.Size,
				})
			}

			if err := formatter.Write(r.File); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}
}

func buildMatchers(cmd *cobra.Command) []matcher.Matcher {
	var matchers []matcher.Matcher

	regex, _ := cmd.Flags().GetString("regex")
	if regex != "" {
		m, err := matcher.NewRegexMatcher(regex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid regex: %v\n", err)
			os.Exit(1)
		}
		matchers = append(matchers, m)
	}

	regexPath, _ := cmd.Flags().GetString("regex-path")
	if regexPath != "" {
		m, err := matcher.NewRegexPathMatcher(regexPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid path regex: %v\n", err)
			os.Exit(1)
		}
		matchers = append(matchers, m)
	}

	entropyThreshold, _ := cmd.Flags().GetFloat64("entropy")
	if entropyThreshold > 0 {
		matchers = append(matchers, matcher.NewEntropyMatcher(
			matcher.WithEntropyThreshold(entropyThreshold),
		))
	}

	mimeTypes, _ := cmd.Flags().GetStringSlice("mime")
	if len(mimeTypes) > 0 {
		matchers = append(matchers, matcher.NewMIMEMatcher(mimeTypes))
	}

	exts, _ := cmd.Flags().GetStringSlice("ext")
	if len(exts) > 0 {
		matchers = append(matchers, matcher.NewExtensionMatcher(exts))
	}

	glob, _ := cmd.Flags().GetString("glob")
	if glob != "" {
		matchers = append(matchers, matcher.NewGlobMatcher(glob))
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

	modifiedFilter, _ := cmd.Flags().GetString("modified")
	if modifiedFilter != "" {
		dur, err := matcher.ParseDuration(modifiedFilter[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid duration: %v\n", err)
			os.Exit(1)
		}
		matchers = append(matchers, matcher.NewModifiedMatcher(modifiedFilter[:1], dur))
	}

	empty, _ := cmd.Flags().GetBool("empty")
	if empty {
		matchers = append(matchers, matcher.NewEmptyMatcher())
	}

	return matchers
}

func detectSource(cmd *cobra.Command, paths []string, workers int, excludes []string, follow bool) (source.Source, error) {
	if len(paths) == 0 {
		return nil, scanner.ErrNoPath
	}

	path := paths[0]

	if strings.HasPrefix(path, "s3://") {
		s3Region, _ := cmd.Flags().GetString("s3-region")
		s3Endpoint, _ := cmd.Flags().GetString("s3-endpoint")

		bucket, prefix, err := source.ParseS3Path(path)
		if err != nil {
			return nil, err
		}

		opts := []source.S3Option{
			source.WithS3Region(s3Region),
			source.WithS3Endpoint(s3Endpoint),
			source.WithS3Workers(workers),
		}

		return source.NewS3Source(bucket, prefix, opts...), nil
	}

	if strings.HasPrefix(path, "git://") || source.IsGitRepo(path) {
		gitModeStr, _ := cmd.Flags().GetString("git-mode")
		gitBranch, _ := cmd.Flags().GetString("git-branch")
		gitSinceStr, _ := cmd.Flags().GetString("git-since")

		var gitMode source.GitMode
		if gitModeStr == "full" {
			gitMode = source.GitModeFull
		} else {
			gitMode = source.GitModeWorking
		}

		opts := []source.GitOption{
			source.WithGitMode(gitMode),
			source.WithGitBranch(gitBranch),
			source.WithGitWorkers(workers),
		}

		if gitSinceStr != "" {
			since, err := time.Parse("2006-01-02", gitSinceStr)
			if err != nil {
				return nil, fmt.Errorf("invalid git-since date: %v", err)
			}
			opts = append(opts, source.WithGitSince(since))
		}

		gitPath := strings.TrimPrefix(path, "git://")
		gitPath = strings.TrimPrefix(gitPath, "file://")

		return source.NewGitSource(gitPath, opts...), nil
	}

	lsOpts := []source.LocalOption{
		source.WithWorkers(workers),
		source.WithExcludes(excludes...),
		source.WithFollowLinks(follow),
	}

	return source.NewLocalSource(append(lsOpts, source.WithPaths(paths...))...), nil
}
