package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
)

type StackFrame struct {
	Function string
	Filename string
	Line     int
	Text     string
}

func (sf StackFrame) GetLinesAround(contextLines int) ([]string, error) {
	// Open the file for reading
	file, err := os.Open(sf.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Create a new scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Find the starting and ending lines
	startLine := sf.getLineNum() - contextLines
	if startLine < 1 {
		startLine = 1
	}
	endLine := sf.getLineNum() + contextLines

	// Read the lines within the range
	lines := []string{}
	for i := 1; scanner.Scan() && i <= endLine; i++ {
		if i >= startLine && i <= endLine {
			lines = append(lines, scanner.Text())
		}

		if i > endLine {
			break
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	return lines, nil
}

func (sf StackFrame) getLineNum() int {
	return sf.Line
}

func extractStackTraceInfo(stackTrace []byte) []*StackFrame {
	// Compile the regular expression pattern to match each stack frame
	pattern := regexp.MustCompile(`(\S+):(\d+)`)

	// split stackTrace into individual lines
	lines := strings.Split(string(stackTrace), "\n")

	// Create a new slice of StackFrame structs
	stackFrames := make([]*StackFrame, 0)

	for _, line := range lines {
		// Find all matches in the stack trace
		matches := pattern.FindAllSubmatch([]byte(line), -1)

		// Append a new StackFrame instance for each match
		for _, match := range matches {
			// parse match[3] as an int
			lineNum, err := strconv.Atoi(string(match[2]))
			if err != nil {
				log.Warn().Err(err).Msg("failed to parse line number")
				continue
			}

			stackFrames = append(stackFrames, &StackFrame{
				Function: "",
				Filename: string(match[1]),
				Line:     lineNum,
			})
		}
	}

	return stackFrames
}

// callPinocchioCrash calls the CLI too `pinocchio mine go-crash -` and pass in stackTrace as stdin.
func callPinocchioCrash(stackTrace string) {
	stackFrames := extractStackTraceInfo([]byte(stackTrace))
	for _, sf := range stackFrames {
		// read in the lines around and put them into Text
		lines, err := sf.GetLinesAround(10)
		if err != nil {
			log.Warn().Err(err).Msg("failed to get lines around")
			continue
		}
		sf.Text = strings.Join(lines, "\n")
	}

	// export stackFrames to a temporary file in JSON format
	tmpFile := "/tmp/stackframes.json"
	f, err := os.Create(tmpFile)
	if err != nil {
		log.Warn().Err(err).Msg("failed to create temporary file")
		return
	}
	defer f.Close()

	err = exportStackFrames(f, stackFrames)
	if err != nil {
		log.Warn().Err(err).Msg("failed to export stack frames")
		return
	}
	log.Info().Msgf("exported stack frames to %s", tmpFile)

	runPinocchio(stackTrace)
}

func runPinocchio(stackTrace string) {
	cmd := exec.Command("pinocchio", "mine", "go-crash", "--stack-frames", "/tmp/stackFrames.json", "--stacktrace-file", "-")
	cmd.Stdin = strings.NewReader(stackTrace)

	// Use StdoutPipe to create a pipe that will be used to stream the output of the command
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating pipe for stdout: %v\n", err)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting pinocchio command: %v\n", err)
		return
	}

	// Create a reader to read from the pipe
	reader := bufio.NewReader(stdoutPipe)

	// Continuously read from the pipe until the command completes
	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Error reading from pipe: %v\n", err)
			}
			break
		}
		fmt.Print(string(r))
	}

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Error running pinocchio command: %v\n", err)
	}
}

func exportStackFrames(f *os.File, frames []*StackFrame) error {
	err := json.NewEncoder(f).Encode(frames)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var stackTrace []byte
	defer func() {
		if err := recover(); err != nil {
			stackTrace = debug.Stack()
			fmt.Printf("panic at the disgotheque: %v\n%s", err, stackTrace)

			callPinocchioCrash(string(stackTrace))
		}
	}()

	var a *int
	*a = 1
}
