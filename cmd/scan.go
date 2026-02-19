package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ripkitten-co/filehound/internal/matcher"
	"github.com/ripkitten-co/filehound/internal/output"
	"github.com/ripkitten-co/filehound/internal/scanner"
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
	scanCmd.Flags().StringP("output", "o", "", "output format: table, json, csv (default: table)")
	scanCmd.Flags().String("out-file", "", "write output to file instead of stdout")
	scanCmd.Flags().Bool("no-header", false, "omit header row in table/CSV output")

	viper.BindPFlags(scanCmd.Flags())
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

	opts := []scanner.Option{
		scanner.WithPaths(paths...),
		scanner.WithWorkers(workers),
		scanner.WithExcludes(excludes...),
		scanner.WithFollowLinks(follow),
	}

	s := scanner.New(opts...)

	results, err := s.Scan()
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
	defer formatter.End()

	for {
		select {
		case <-ctx.Done():
			return
		case r, ok := <-results:
			if !ok {
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
