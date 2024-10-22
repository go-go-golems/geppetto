I want to build a file walker to easily compute statistics over a set of files / directories.

For that I want to either get the files from walking an actual directory, but I also want to input
a set of filenames (after filtering, for example), and then still have the walker decompose that into directories.

The ast walker should be able to do a pre and post visit node.

Node can be a file, and you get the file metadata using FileData for example.

The node callbacks get the node type and node information, and the parent, a list of the children (recursively) and a list of the immediate children.

Further more, the walker also exposes the information to get the information for a Node itself, so that I can work across tree boundaries.

---
package github.com/go-go-golems/glazed/pkg/cmds/parameters

type FileData struct {
	Content          string
	ParsedContent    interface{}
	ParseError       error
	RawContent       []byte
	StringContent    string
	IsList           bool
	IsObject         bool
	BaseName         string
	Extension        string
	FileType         FileType
	Path             string
	RelativePath     string
	AbsolutePath     string
	Size             int64
	LastModifiedTime time.Time
	Permissions      os.FileMode
	IsDirectory      bool
}

---

Design a go package for that AST walker.