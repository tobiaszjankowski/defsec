package parser

import (
	"context"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/aquasecurity/defsec/pkg/providers/dockerfile"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"golang.org/x/xerrors"
)

type Parser struct {
	skipRequired bool
}

const requiredFile = "Dockerfile"

// New creates a new Dockerfile parser
func New(options ...Option) *Parser {
	p := &Parser{}
	for _, option := range options {
		option(p)
	}
	return p
}

func (p *Parser) ParseFS(ctx context.Context, target fs.FS, path string) (map[string]*dockerfile.Dockerfile, error) {

	files := make(map[string]*dockerfile.Dockerfile)
	if err := fs.WalkDir(target, filepath.ToSlash(path), func(path string, entry fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !p.Required(path) {
			return nil
		}
		df, err := p.ParseFile(ctx, target, path)
		if err != nil {
			// TODO add debug for parse errors
			return nil
		}
		files[path] = df
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

// ParseFile parses Dockerfile content from the provided filesystem path.
func (p *Parser) ParseFile(_ context.Context, fs fs.FS, path string) (*dockerfile.Dockerfile, error) {
	f, err := fs.Open(filepath.ToSlash(path))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return p.parse(path, f)
}

func (p *Parser) Required(path string) bool {
	if p.skipRequired {
		return true
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if strings.EqualFold(base, requiredFile+ext) {
		return true
	}
	if strings.EqualFold(ext, "."+requiredFile) {
		return true
	}
	return false
}

func (p *Parser) parse(path string, r io.Reader) (*dockerfile.Dockerfile, error) {
	parsed, err := parser.Parse(r)
	if err != nil {
		return nil, xerrors.Errorf("dockerfile parse error: %w", err)
	}

	var parsedFile dockerfile.Dockerfile
	parsedFile.Stages = make(map[string][]dockerfile.Command)

	var stageIndex int
	fromValue := "args"
	for _, child := range parsed.AST.Children {
		child.Value = strings.ToLower(child.Value)

		instr, err := instructions.ParseInstruction(child)
		if err != nil {
			return nil, xerrors.Errorf("process dockerfile instructions: %w", err)
		}

		if _, ok := instr.(*instructions.Stage); ok {
			if fromValue != "args" {
				stageIndex++
			}
			fromValue = strings.TrimSpace(strings.TrimPrefix(child.Original, "FROM "))
		}

		cmd := dockerfile.Command{
			Cmd:       child.Value,
			Original:  child.Original,
			Flags:     child.Flags,
			Stage:     stageIndex,
			Path:      path,
			StartLine: child.StartLine,
			EndLine:   child.EndLine,
		}

		if child.Next != nil && len(child.Next.Children) > 0 {
			cmd.SubCmd = child.Next.Children[0].Value
			child = child.Next.Children[0]
		}

		cmd.JSON = child.Attributes["json"]
		for n := child.Next; n != nil; n = n.Next {
			cmd.Value = append(cmd.Value, n.Value)
		}

		parsedFile.Stages[fromValue] = append(parsedFile.Stages[fromValue], cmd)

	}

	return &parsedFile, nil
}
