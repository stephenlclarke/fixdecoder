package decoder

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
)

var (
	loadDictionary   = LoadDictionary
	parseFix         = ParseFix
	streamLogFunc    = streamLog
	getTermSize      = term.GetSize // allow override in tests
	enableValidation = false        // controlled by -validate flag
)

var (
	ColourReset = "\033[0m"
	ColourLine  = "\033[38;5;244m"
	ColourTag   = "\033[38;5;81m"
	ColourName  = "\033[38;5;151m"
	ColourValue = "\033[38;5;228m"
	ColourEnum  = "\033[38;5;214m"
	ColourFile  = "\033[95m"
	ColourError = "\033[31m"
	ColourMsg   = "\033[97m"
	ColourTitle = "\033[31m"
)

func DisableColours() {
	ColourReset = ""
	ColourLine = ""
	ColourTag = ""
	ColourName = ""
	ColourValue = ""
	ColourEnum = ""
	ColourFile = ""
	ColourError = ""
	ColourMsg = ""
	ColourTitle = ""
}

func PrettifySimple(msg string) string {
	dict := loadDictionary(msg)
	return Prettify(msg, dict)
}

func Prettify(msg string, dict *FixTagLookup) string {
	var sb strings.Builder

	for _, fv := range parseFix(msg) {
		name := dict.GetFieldName(fv.Tag)
		desc := dict.GetEnumDescription(fv.Tag, fv.Value)

		sb.WriteString(fmt.Sprintf("    %s%4d%s (%s%s%s): %s%s%s",
			ColourTag, fv.Tag, ColourReset,
			ColourName, name, ColourReset,
			ColourValue, fv.Value, ColourReset,
		))

		if desc != "" {
			sb.WriteString(fmt.Sprintf(" (%s%s%s)", ColourEnum, desc, ColourReset))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func PrettifyFiles(paths []string, out io.Writer, errOut io.Writer) int {
	hadError := false

	// If no paths at all, default to stdin (unchanged behaviour)
	if len(paths) == 0 {
		if err := streamLogFunc(os.Stdin, out); err != nil {
			fmt.Fprintln(errOut, ColourError+"Error reading input:"+err.Error()+ColourReset)
			return 1
		}

		return 0
	}

	// Otherwise, iterate over every supplied path.
	// Treat the single dash "-" as a synonym for stdin.
	for _, path := range paths {
		var (
			r   io.Reader
			c   io.Closer // nil when reading stdin
			err error
		)

		if path == "-" {
			fmt.Fprint(out, "Processing: (stdin)\n\n")
			r = os.Stdin // read from pipe/tty
		} else {
			fmt.Fprint(out, "Processing: ", ColourFile, path, ColourReset, "\n\n")

			var f *os.File
			f, err = os.Open(path)
			if err != nil {
				fmt.Fprintln(errOut, ColourError+"Cannot open file:"+err.Error()+ColourReset)
				hadError = true
				continue
			}

			r, c = f, f // will close after streaming
		}

		if err = streamLogFunc(r, out); err != nil {
			fmt.Fprintln(errOut, ColourError+"Error reading file:"+err.Error()+ColourReset)
			hadError = true
		}

		if c != nil {
			c.Close()
		}
	}

	if hadError {
		return 1
	}

	return 0
}

func streamLog(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	termWidth := getTerminalWidth()
	separator := ColourTitle + strings.Repeat("=", termWidth) + ColourReset + "\n"

	for scanner.Scan() {
		line := scanner.Text()
		handleLogLine(line, out, separator)
	}

	return scanner.Err()
}

func handleLogLine(line string, out io.Writer, separator string) {
	matches := findFixMessageIndices(line)

	if len(matches) == 0 {
		fmt.Fprint(out, ColourLine, line, ColourReset, "\n")
		return
	}

	fixMessages, colouredLine := extractFixMessagesAndFormat(line, matches)
	fmt.Fprint(out, colouredLine)
	fmt.Fprint(out, separator)

	for _, msg := range fixMessages {
		processFixMessage(msg, out, separator)
	}
}

func processFixMessage(msg string, out io.Writer, separator string) {
	dict := loadDictionary(msg)
	fmt.Fprint(out, Prettify(msg, dict))

	// Validation
	if enableValidation {
		errors := ValidateFixMessage(msg, dict)
		if len(errors) > 0 {
			fmt.Fprint(out, separator)

			for _, err := range errors {
				fmt.Fprintf(out, "%s== %s%s\n", ColourError, err, ColourReset)
			}
		}
	}

	fmt.Fprint(out, separator)
}

func getTerminalWidth() int {
	if w, _, err := getTermSize(int(os.Stdout.Fd())); err == nil {
		return w
	}
	return 80
}

func findFixMessageIndices(line string) [][]int {
	re := regexp.MustCompile(`8=FIX.*?10=\d{3}\x01`)
	return re.FindAllStringIndex(line, -1)
}

func extractFixMessagesAndFormat(line string, matches [][]int) ([]string, string) {
	var (
		output      strings.Builder
		lastIndex   int
		fixMessages []string
	)

	for _, match := range matches {
		start, end := match[0], match[1]
		before := line[lastIndex:start]
		fixPart := line[start:end]

		output.WriteString(ColourLine + before + ColourMsg + fixPart)
		fixMessages = append(fixMessages, fixPart)
		lastIndex = end
	}

	// Append remaining part of the line after last FIX message
	output.WriteString(ColourLine + line[lastIndex:] + ColourReset + "\n")

	return fixMessages, output.String()
}

func SetValidation(enabled bool) {
	enableValidation = enabled
}
