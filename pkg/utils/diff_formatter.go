package utils

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)


type DiffStats struct {
	FilesChanged int `json:"files_changed"`
	LinesAdded   int `json:"lines_added"`
	LinesRemoved int `json:"lines_removed"`
}


const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorBold   = "\033[1m"
)


const (
	DiffHeaderPrefix    = "diff --git"
	DiffIndexPrefix     = "index "
	DiffFileFromPrefix  = "--- "
	DiffFileToPrefix    = "+++ "
	DiffHunkPrefix      = "@@"
	DiffAddPrefix       = "+"
	DiffDelPrefix       = "-"
)


func ExtractChangedFiles(diff string) []string {
	files := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(diff))
	
	for scanner.Scan() {
		line := scanner.Text()
		

		if strings.HasPrefix(line, DiffHeaderPrefix) {

			parts := strings.Fields(line)
			if len(parts) >= 4 {

				fileA := strings.TrimPrefix(parts[2], "a/")
				fileB := strings.TrimPrefix(parts[3], "b/")
				

				if fileB != "/dev/null" {
					files = append(files, fileB)
				} else if fileA != "/dev/null" {

					files = append(files, fileA)
				}
			}
		}
	}
	
	return files
}


func FilterDiffByFile(diff, targetFile string) string {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(diff))
	inTargetFile := false
	
	for scanner.Scan() {
		line := scanner.Text()
		

		if strings.HasPrefix(line, DiffHeaderPrefix) {

			inTargetFile = strings.Contains(line, targetFile)
			if inTargetFile {
				result.WriteString(line + "\n")
			}
		} else if inTargetFile {
			result.WriteString(line + "\n")
			


		}
	}
	
	return result.String()
}


func CleanDiffForPatch(diff string) string {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(diff))
	inDiffContent := false
	
	for scanner.Scan() {
		line := scanner.Text()
		

		if !inDiffContent && strings.HasPrefix(line, DiffHeaderPrefix) {
			inDiffContent = true
		}
		

		if inDiffContent && (strings.HasPrefix(line, DiffHeaderPrefix) ||
			strings.HasPrefix(line, DiffIndexPrefix) ||
			strings.HasPrefix(line, DiffFileFromPrefix) ||
			strings.HasPrefix(line, DiffFileToPrefix) ||
			strings.HasPrefix(line, DiffHunkPrefix) ||
			strings.HasPrefix(line, DiffAddPrefix) ||
			strings.HasPrefix(line, DiffDelPrefix) ||
			(!strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "@"))) {
			
			result.WriteString(line + "\n")
		}
	}
	
	return result.String()
}


func FormatDiff(diff string, useColors bool) string {
	if !useColors {
		return diff
	}
	
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(diff))
	
	for scanner.Scan() {
		line := scanner.Text()
		formattedLine := formatDiffLine(line)
		result.WriteString(formattedLine + "\n")
	}
	
	return result.String()
}


func formatDiffLine(line string) string {
	switch {
	case strings.HasPrefix(line, DiffHeaderPrefix):
		return ColorBlue + ColorBold + line + ColorReset
	case strings.HasPrefix(line, DiffIndexPrefix):
return line
	case strings.HasPrefix(line, DiffFileFromPrefix) || strings.HasPrefix(line, DiffFileToPrefix):
		return ColorBold + line + ColorReset
	case strings.HasPrefix(line, DiffHunkPrefix):
		return ColorCyan + line + ColorReset
	case strings.HasPrefix(line, DiffAddPrefix):
		return ColorGreen + line + ColorReset
	case strings.HasPrefix(line, DiffDelPrefix):
		return ColorRed + line + ColorReset
	default:
		return line
	}
}


func CalculateDiffStats(diff string) DiffStats {
	stats := DiffStats{}
	scanner := bufio.NewScanner(strings.NewReader(diff))
	
	filesSet := make(map[string]bool)
	
	for scanner.Scan() {
		line := scanner.Text()
		

		if strings.HasPrefix(line, DiffHeaderPrefix) {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				fileA := strings.TrimPrefix(parts[2], "a/")
				fileB := strings.TrimPrefix(parts[3], "b/")
				
				if fileB != "/dev/null" {
					filesSet[fileB] = true
				} else if fileA != "/dev/null" {
					filesSet[fileA] = true
				}
			}
		}
		

		if strings.HasPrefix(line, DiffAddPrefix) && !strings.HasPrefix(line, "+++") {
			stats.LinesAdded++
		} else if strings.HasPrefix(line, DiffDelPrefix) && !strings.HasPrefix(line, "---") {
			stats.LinesRemoved++
		}
	}
	
	stats.FilesChanged = len(filesSet)
	return stats
}


func IsTerminal(f *os.File) bool {
	fd := f.Fd()
	_, _, err := getTerminalSize(int(fd))
	return err == nil
}


func getTerminalSize(fd int) (width, height int, err error) {
	var dimensions [4]uint16
	
	if _, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&dimensions)),
		0, 0, 0); errno != 0 {
		return 0, 0, errno
	}
	
	return int(dimensions[1]), int(dimensions[0]), nil
}


func FormatFilePath(fromPath, toPath string) string {

	if fromPath != toPath && fromPath != "/dev/null" && toPath != "/dev/null" {
		return fmt.Sprintf("%s → %s", fromPath, toPath)
	}
	

	if fromPath == "/dev/null" {
		return toPath + " (new)"
	}
	

	if toPath == "/dev/null" {
		return fromPath + " (deleted)"
	}
	
	return toPath
}


func ParseHunkHeader(line string) (fromStart, fromCount, toStart, toCount int, err error) {

	re := regexp.MustCompile(`@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
	matches := re.FindStringSubmatch(line)
	
	if len(matches) == 0 {
		return 0, 0, 0, 0, fmt.Errorf("invalid hunk header: %s", line)
	}
	
	fromStart, _ = strconv.Atoi(matches[1])
	if matches[2] != "" {
		fromCount, _ = strconv.Atoi(matches[2])
	} else {
		fromCount = 1
	}
	
	toStart, _ = strconv.Atoi(matches[3])
	if matches[4] != "" {
		toCount, _ = strconv.Atoi(matches[4])
	} else {
		toCount = 1
	}
	
	return fromStart, fromCount, toStart, toCount, nil
}


func IsBinaryFile(diffSection string) bool {
	return strings.Contains(diffSection, "Binary files") ||
		strings.Contains(diffSection, "GIT binary patch")
}


func SplitDiffIntoFiles(diff string) map[string]string {
	files := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(diff))
	
	var currentFile string
	var currentDiff strings.Builder
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.HasPrefix(line, DiffHeaderPrefix) {

			if currentFile != "" {
				files[currentFile] = currentDiff.String()
			}
			

			parts := strings.Fields(line)
			if len(parts) >= 4 {
				fileB := strings.TrimPrefix(parts[3], "b/")
				if fileB == "/dev/null" {
					currentFile = strings.TrimPrefix(parts[2], "a/")
				} else {
					currentFile = fileB
				}
			}
			
			currentDiff.Reset()
			currentDiff.WriteString(line + "\n")
		} else {
			currentDiff.WriteString(line + "\n")
		}
	}
	

	if currentFile != "" {
		files[currentFile] = currentDiff.String()
	}
	
	return files
}
