package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ripkitten-co/filehound/internal/hash"
	"github.com/ripkitten-co/filehound/internal/matcher"
	"github.com/ripkitten-co/filehound/internal/scanner"
	"github.com/spf13/cobra"
)

var hashCmd = &cobra.Command{
	Use:   "hash [PATH...]",
	Short: "Compute file hashes and find duplicates",
	Long: `Compute file hashes and optionally find duplicate files.

Examples:
  # Hash all files in current directory
  filehound hash .

  # Find duplicate files
  filehound hash . --duplicates

  # Use SHA1 instead of SHA256
  filehound hash . --algorithm sha1

  # Hash only specific files
  filehound hash . --ext .go --output json`,
	Args: cobra.MinimumNArgs(0),
	Run:  runHash,
}

func init() {
	rootCmd.AddCommand(hashCmd)

	hashCmd.Flags().StringP("algorithm", "a", "sha256", "hash algorithm: sha1, sha256, sha512")
	hashCmd.Flags().BoolP("duplicates", "d", false, "find and display duplicate files only")
	hashCmd.Flags().StringP("glob", "g", "", "glob pattern for filename")
	hashCmd.Flags().StringSlice("ext", []string{}, "file extensions to match")
	hashCmd.Flags().String("size", "", "size filter")
	hashCmd.Flags().IntP("workers", "w", 0, "number of parallel workers")
	hashCmd.Flags().StringP("output", "o", "", "output format: table, json, csv (default: table)")
	hashCmd.Flags().String("out-file", "", "write output to file instead of stdout")
	hashCmd.Flags().Bool("no-header", false, "omit header row in table/CSV output")
}

func runHash(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	algorithm, _ := cmd.Flags().GetString("algorithm")
	findDuplicates, _ := cmd.Flags().GetBool("duplicates")
	workers, _ := cmd.Flags().GetInt("workers")
	if workers <= 0 {
		workers = 8
	}

	var alg hash.Algorithm
	switch algorithm {
	case "sha1":
		alg = hash.SHA1
	case "sha256":
		alg = hash.SHA256
	case "sha512":
		alg = hash.SHA512
	default:
		fmt.Fprintf(os.Stderr, "unknown algorithm: %s (use sha1, sha256, or sha512)\n", algorithm)
		os.Exit(1)
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

	matchers := buildHashMatchers(cmd)

	var filePaths []string
	for {
		select {
		case <-ctx.Done():
			return
		case r, ok := <-results:
			if !ok {
				goto process
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

			filePaths = append(filePaths, r.File.Path)
		}
	}

process:
	hasher := hash.NewHasher(hash.WithAlgorithm(alg))
	hashResults := hasher.HashFiles(filePaths)

	var allResults []hash.Result
	for r := range hashResults {
		allResults = append(allResults, r)
	}

	if findDuplicates {
		printDuplicates(allResults, cmd)
	} else {
		printHashResults(allResults, cmd)
	}
}

func buildHashMatchers(cmd *cobra.Command) []matcher.Matcher {
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

func printHashResults(results []hash.Result, cmd *cobra.Command) {
	outputFormat, _ := cmd.Flags().GetString("output")
	noHeader, _ := cmd.Flags().GetBool("no-header")

	if outputFormat == "json" {
		for _, r := range results {
			fmt.Printf(`{"path":"%s","hash":"%s","algorithm":"%s","size":%d}`+"\n", r.Path, r.Hash, r.Algorithm, r.Size)
		}
		return
	}

	if outputFormat == "csv" {
		if !noHeader {
			fmt.Println("path,hash,algorithm,size")
		}
		for _, r := range results {
			fmt.Printf("%s,%s,%s,%d\n", r.Path, r.Hash, r.Algorithm, r.Size)
		}
		return
	}

	for _, r := range results {
		fmt.Printf("%s  %s\n", r.Hash[:16], r.Path)
	}
}

func printDuplicates(results []hash.Result, cmd *cobra.Command) {
	duplicates := hash.FindDuplicates(results)

	if len(duplicates) == 0 {
		fmt.Println("No duplicate files found")
		return
	}

	outputFormat, _ := cmd.Flags().GetString("output")

	if outputFormat == "json" {
		for _, group := range duplicates {
			fmt.Printf(`{"hash":"%s","size":%d,"count":%d,"files":[`, group.Hash, group.Size, len(group.Files))
			for i, f := range group.Files {
				if i > 0 {
					fmt.Printf(",")
				}
				fmt.Printf(`"%s"`, f)
			}
			fmt.Println("]}")
		}
		return
	}

	fmt.Printf("Found %d groups of duplicate files:\n\n", len(duplicates))
	for i, group := range duplicates {
		fmt.Printf("Group %d (hash: %s, size: %d bytes, %d files):\n", i+1, group.Hash[:16]+"...", group.Size, len(group.Files))
		for _, f := range group.Files {
			fmt.Printf("  - %s\n", f)
		}
		fmt.Println()
	}
}
