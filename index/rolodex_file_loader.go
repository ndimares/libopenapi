// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"time"
)

type LocalFS struct {
	entryPointDirectory string
	baseDirectory       string
	Files               map[string]RolodexFile
	parseTime           int64
	logger              *slog.Logger
	readingErrors       []error
	filters             []string
}

func (l *LocalFS) GetFiles() map[string]RolodexFile {
	return l.Files
}

func (l *LocalFS) Open(name string) (fs.File, error) {
	if !filepath.IsAbs(name) {
		var absErr error
		name, absErr = filepath.Abs(filepath.Join(l.baseDirectory, name))
		if absErr != nil {
			return nil, absErr
		}
	}

	if f, ok := l.Files[name]; ok {
		return &localRolodexFile{f: f}, nil
	} else {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
}

type LocalFile struct {
	filename      string
	name          string
	extension     FileExtension
	data          []byte
	fullPath      string
	lastModified  time.Time
	readingErrors []error
	index         *SpecIndex
	parsed        *yaml.Node
}

func (l *LocalFile) GetIndex() *SpecIndex {
	return l.index
}

func (l *LocalFile) Index(config *SpecIndexConfig) (*SpecIndex, error) {
	if l.index != nil {
		return l.index, nil
	}
	content := l.data

	// first, we must parse the content of the file
	info, err := datamodel.ExtractSpecInfoWithDocumentCheck(content, true)
	if err != nil {
		return nil, err
	}

	index := NewSpecIndexWithConfig(info.RootNode, config)
	index.specAbsolutePath = l.fullPath
	l.index = index
	return index, nil

}

func (l *LocalFile) GetContent() string {
	return string(l.data)
}

func (l *LocalFile) GetContentAsYAMLNode() (*yaml.Node, error) {
	if l.parsed != nil {
		return l.parsed, nil
	}
	if l.index != nil && l.index.root != nil {
		return l.index.root, nil
	}
	if l.data == nil {
		return nil, fmt.Errorf("no data to parse for file: %s", l.fullPath)
	}
	var root yaml.Node
	err := yaml.Unmarshal(l.data, &root)
	if err != nil {
		return nil, err
	}
	if l.index != nil && l.index.root == nil {
		l.index.root = &root
	}
	l.parsed = &root
	return &root, nil
}

func (l *LocalFile) GetFileExtension() FileExtension {
	return l.extension
}

func (l *LocalFile) GetFullPath() string {
	return l.fullPath
}

func (l *LocalFile) GetErrors() []error {
	return l.readingErrors
}

type LocalFSConfig struct {
	// the base directory to index
	BaseDirectory string
	FileFilters   []string
	DirFS         fs.FS
}

func NewLocalFSWithConfig(config *LocalFSConfig) (*LocalFS, error) {
	localFiles := make(map[string]RolodexFile)
	var allErrors []error

	// if the basedir is an absolute file, we're just going to index that file.
	ext := filepath.Ext(config.BaseDirectory)
	file := filepath.Base(config.BaseDirectory)

	var absBaseDir string
	var absBaseErr error

	absBaseDir, absBaseErr = filepath.Abs(config.BaseDirectory)

	if absBaseErr != nil {
		return nil, absBaseErr
	}

	walkErr := fs.WalkDir(config.DirFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// we don't care about directories, or errors, just read everything we can.
		if d == nil || d.IsDir() {
			return nil
		}

		if len(ext) > 2 && p != file {
			return nil
		}

		if strings.HasPrefix(p, ".") {
			return nil
		}

		if len(config.FileFilters) > 0 {
			if !slices.Contains(config.FileFilters, p) {
				return nil
			}
		}

		extension := ExtractFileType(p)
		var readingErrors []error
		abs, absErr := filepath.Abs(filepath.Join(config.BaseDirectory, p))
		if absErr != nil {
			readingErrors = append(readingErrors, absErr)
			logger.Error("cannot create absolute path for file: ", "file", p, "error", absErr.Error())
		}

		var fileData []byte

		switch extension {
		case YAML, JSON:

			file, readErr := config.DirFS.Open(p)
			modTime := time.Now()
			if readErr != nil {
				readingErrors = append(readingErrors, readErr)
				allErrors = append(allErrors, readErr)
				logger.Error("[rolodex] cannot open file: ", "file", abs, "error", readErr.Error())
				return nil
			}
			stat, statErr := file.Stat()
			if statErr != nil {
				readingErrors = append(readingErrors, statErr)
				allErrors = append(allErrors, statErr)
				logger.Error("[rolodex] cannot stat file: ", "file", abs, "error", statErr.Error())
			}
			if stat != nil {
				modTime = stat.ModTime()
			}
			fileData, readErr = io.ReadAll(file)
			if readErr != nil {
				readingErrors = append(readingErrors, readErr)
				allErrors = append(allErrors, readErr)
				logger.Error("cannot read file data: ", "file", abs, "error", readErr.Error())
				return nil
			}

			logger.Debug("collecting JSON/YAML file", "file", abs)
			localFiles[abs] = &LocalFile{
				filename:      p,
				name:          filepath.Base(p),
				extension:     ExtractFileType(p),
				data:          fileData,
				fullPath:      abs,
				lastModified:  modTime,
				readingErrors: readingErrors,
			}
		case UNSUPPORTED:
			logger.Debug("skipping non JSON/YAML file", "file", abs)
		}
		return nil
	})

	if walkErr != nil {
		return nil, walkErr
	}

	return &LocalFS{
		Files:               localFiles,
		logger:              logger,
		baseDirectory:       absBaseDir,
		entryPointDirectory: config.BaseDirectory,
		readingErrors:       allErrors,
	}, nil
}

func NewLocalFS(baseDir string, dirFS fs.FS) (*LocalFS, error) {
	config := &LocalFSConfig{
		BaseDirectory: baseDir,
		DirFS:         dirFS,
	}
	return NewLocalFSWithConfig(config)
}

func (l *LocalFile) FullPath() string {
	return l.fullPath
}

func (l *LocalFile) Name() string {
	return l.name
}

func (l *LocalFile) Size() int64 {
	return int64(len(l.data))
}

func (l *LocalFile) Mode() fs.FileMode {
	return fs.FileMode(0)
}

func (l *LocalFile) ModTime() time.Time {
	return l.lastModified
}

func (l *LocalFile) IsDir() bool {
	return false
}

func (l *LocalFile) Sys() interface{} {
	return nil
}

type localRolodexFile struct {
	f      RolodexFile
	offset int64
}

func (r *localRolodexFile) Close() error {
	return nil
}

func (r *localRolodexFile) Stat() (fs.FileInfo, error) {
	return r.f, nil
}

func (r *localRolodexFile) Read(b []byte) (int, error) {
	if r.offset >= int64(len(r.f.GetContent())) {
		return 0, io.EOF
	}
	if r.offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: r.f.GetFullPath(), Err: fs.ErrInvalid}
	}
	n := copy(b, r.f.GetContent()[r.offset:])
	r.offset += int64(n)
	return n, nil
}
