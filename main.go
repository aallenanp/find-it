package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/windows"
)

// Return values for GetDriveType.
const (
	DRIVE_UNKNOWN     = 0
	DRIVE_NO_ROOT_DIR = 1
	DRIVE_REMOVABLE   = 2
	DRIVE_FIXED       = 3 // Fixed Drive
	DRIVE_REMOTE      = 4 // Network Share
	DRIVE_CDROM       = 5
	DRIVE_RAMDISK     = 6
)

// getAvailableDrives scans A-Z and returns every local root path that exists.
// Mapped network drives (DRIVE_REMOTE) are intentionally excluded.
func getAvailableDrives() []string {
	var drives []string
	for c := 'A'; c <= 'Z'; c++ {
		root := string(c) + ":\\"
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		// Skip mapped network drives.
		rootPtr, err := windows.UTF16PtrFromString(root)
		if err == nil && windows.GetDriveType(rootPtr) == DRIVE_REMOTE {
			continue
		}
		drives = append(drives, root)
	}
	return drives
}

// matchName performs case-insensitive glob matching using filepath.Match.
// Supports patterns like: prefix*, *infix*, *suffix, or exact.
func matchName(pattern, name string) bool {
	matched, err := filepath.Match(strings.ToLower(pattern), strings.ToLower(name))
	if err != nil {
		// Invalid pattern — treat as no match
		return false
	}
	return matched
}

// search walks root and sends matching paths to the results channel.
// For --type file : checks extension then matches --name against the base name (no ext).
// For --type dir  : matches --name against the directory name.
func search(root, searchType, ext, namePattern string, results chan<- string) {
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Permission errors, unavailable volumes, etc. — skip silently.
			return nil
		}

		// Never try to match root itself.
		if path == root {
			return nil
		}

		entryName := d.Name()

		switch searchType {
		case "file":
			if d.IsDir() {
				return nil
			}
			// Extension check (case-insensitive, strip leading dot).
			fileExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(entryName)), ".")
			if fileExt != strings.ToLower(ext) {
				return nil
			}
			// Name check: match against base name without the extension.
			baseName := strings.TrimSuffix(entryName, filepath.Ext(entryName))
			if !matchName(namePattern, baseName) {
				return nil
			}
			results <- path

		case "dir":
			if !d.IsDir() {
				return nil
			}
			if !matchName(namePattern, entryName) {
				return nil
			}
			results <- path
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "walk error on %s: %v\n", root, err)
	}
}

func printUsageAndExit(errs []string) {
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "  ERROR: %s\n", e)
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  findit --type <file|dir> --name <pattern> --target <path|*> [--ext <ext>]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  findit --type file --ext pst --name *filName* --target C:\\Users")
	fmt.Fprintln(os.Stderr, "  findit --type file --ext pst --name backup* --target *")
	fmt.Fprintln(os.Stderr, "  findit --type dir  --name logs*  --target \"C:\\Program Files\"")
	os.Exit(1)
}

func main() {
	typeFlag := flag.String("type", "", "REQUIRED. Search for 'file' or 'dir'")
	extFlag := flag.String("ext", "", "REQUIRED when --type=file. Extension without dot (e.g. pst, txt)")
	nameFlag := flag.String("name", "", "REQUIRED. Name pattern to match (supports * wildcard, e.g. prefix*, *infix*, *suffix)")
	targetFlag := flag.String("target", "", "REQUIRED. Directory to search (e.g. C:\\Users) or * to search all drives")

	flag.Parse()

	// ── Validation ────────────────────────────────────────────────────────────
	var errs []string

	if *typeFlag == "" {
		errs = append(errs, "--type is required (file or dir)")
	} else if *typeFlag != "file" && *typeFlag != "dir" {
		errs = append(errs, fmt.Sprintf("--type must be 'file' or 'dir', got '%s'", *typeFlag))
	}

	if *typeFlag == "file" && *extFlag == "" {
		errs = append(errs, "--ext is required when --type=file")
	}

	if *nameFlag == "" {
		errs = append(errs, "--name is required")
	}

	if *targetFlag == "" {
		errs = append(errs, "--target is required (path or *)")
	}

	if len(errs) > 0 {
		printUsageAndExit(errs)
	}

	// ── Resolve target(s) ────────────────────────────────────────────────────
	var targets []string

	if *targetFlag == "*" {
		targets = getAvailableDrives()
		if len(targets) == 0 {
			fmt.Fprintln(os.Stderr, "ERROR: no available drives found on this system")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Searching %d drive(s): %s\n\n", len(targets), strings.Join(targets, "  "))
	} else {
		info, err := os.Stat(*targetFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: cannot access --target path '%s': %v\n", *targetFlag, err)
			os.Exit(1)
		}
		if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "ERROR: --target must point to a directory, not a file: '%s'\n", *targetFlag)
			os.Exit(1)
		}
		targets = []string{*targetFlag}
	}

	// ── Search ────────────────────────────────────────────────────────────────
	// Buffered channel so walkers are never blocked waiting for the printer.
	results := make(chan string, 512)
	var wg sync.WaitGroup

	for _, t := range targets {
		wg.Add(1)
		go func(root string) {
			defer wg.Done()
			search(root, *typeFlag, *extFlag, *nameFlag, results)
		}(t)
	}

	// Close the results channel once all goroutines finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Print every result as soon as it arrives.
	count := 0
	for path := range results {
		fmt.Println(path)
		count++
	}

	fmt.Fprintf(os.Stderr, "\nDone | %d match(es) found.\n", count)
}
