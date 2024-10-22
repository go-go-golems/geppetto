# catter

catter is a CLI tool designed to prepare codebase content for Large Language Model (LLM) contexts. It processes
files and directories, prints their contents, and provides detailed token counting statistics. This tool is particularly
useful for developers and researchers working with LLMs who need to analyze and prepare their codebase for model input.

## Features

- Recursively process directories and files
- Filter files based on extensions (include/exclude)
- Match files based on filename or full path using regular expressions
- Limit processing based on individual file size and total processed size
- Count tokens using the tiktoken library (compatible with OpenAI's tokenization)
- Provide configurable levels of token count statistics
- Output file contents to stdout and statistics to stderr
- Option to list files without printing content
- Exclude specific directories
- Respect .gitignore rules (with option to disable)

## Installation

### Prerequisites

- Go 1.16 or higher

### Steps

1. Install the tool:
   ```
   go install github.com/go-go-golems/go-go-labs/cmd/apps/catter@latest
   ```

## Usage

Run the program using the following command:

```
catter [flags] <file1> <directory1> ...
```

### Flags

- `--max-file-size`: Maximum size of individual files in bytes (default: 1MB)
- `--max-total-size`: Maximum total size of all processed files in bytes (default: 10MB)
- `-i, --include`: List of file extensions to include (e.g., .go,.js)
- `-e, --exclude`: List of file extensions to exclude (e.g., .exe,.dll)
- `-s, --stats`: Level of statistics to show: none, total, or detailed (default: none)
- `-f, --match-filename`: List of regular expressions to match filenames
- `-p, --match-path`: List of regular expressions to match full paths
- `-l, --list`: List filenames only without printing content
- `-x, --exclude-dirs`: List of directories to exclude
- `--disable-gitignore`: Disable .gitignore filter
- `-d, --delimiter`: Type of delimiter to use between files: default, xml, markdown, simple, begin-end (default: default)

### Example

```
catter --max-file-size=500000 --max-total-size=5000000 -i .go,.js -e .tmp,.log -s detailed -f "^main" -p "src/" -x vendor,node_modules -d markdown /path/to/your/codebase
```

This command will:
- Process files up to 500KB in size
- Stop after processing a total of 5MB of content
- Only include .go and .js files
- Exclude .tmp and .log files
- Print detailed statistics, including token counts per file and directory
- Only process files with names starting with "main"
- Only process files in paths containing "src/"
- Exclude the "vendor" and "node_modules" directories
- Respect .gitignore rules
- Use markdown-style delimiters between files

## Output

The program outputs two types of information:

1. File contents (stdout): The content of each processed file is printed to standard output, separated by the specified delimiter.

2. Statistics (stderr): Depending on the `-s, --stats` flag:
    - `none`: No statistics are shown (default)
    - `total`: Only the total token count is shown
    - `detailed`: A full summary is printed, including:
        - Total number of files processed
        - Total size processed (in bytes)
        - Total token count
        - Token count per file and directory

## Use Cases

- Preparing codebase for LLM fine-tuning or prompt engineering
- Analyzing token usage across different parts of a codebase
- Generating code summaries for documentation or review purposes
- Preprocessing code for other NLP tasks or tools

## License

This project is licensed under the MIT License.

## Acknowledgements

- [Cobra](https://github.com/spf13/cobra) for CLI interface
- [tiktoken-go](https://github.com/pkoukk/tiktoken-go) for token counting
- [go-gitignore](https://github.com/denormal/go-gitignore) for .gitignore support