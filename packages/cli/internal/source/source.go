// Nome: source.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Implementa adaptadores de entrada usados pela CLI para transformar
// stdin, arquivos e leitura contínua em fontes compatíveis com os contratos de
// ingestão do engine, incluindo controle de linhas e limites de tamanho.
// Package source adapts CLI input streams to the log engine's Source contract.
package source

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"SmokeLab/packages/engine/logs"
)

const DefaultMaxLineBytes = 1024 * 1024

var ErrLineTooLong = errors.New("log line exceeds maximum size")

// Stdin reads complete NDJSON lines from a terminal or a piped input stream.
type Stdin struct {
	reader *lineReader
}

func NewStdin(input io.Reader, maxLineBytes int) (*Stdin, error) {
	reader, err := newLineReader(input, maxLineBytes)
	if err != nil {
		return nil, err
	}
	return &Stdin{reader: reader}, nil
}

func (s *Stdin) Descriptor() logs.SourceDescriptor {
	return logs.SourceDescriptor{ID: "stdin", Name: "stdin", Kind: "stdin"}
}

func (s *Stdin) Next(ctx context.Context) (logs.SourceLine, error) {
	return s.reader.next(ctx)
}

// File reads a file once. A final line without a newline is still emitted.
type File struct {
	file   *os.File
	path   string
	reader *lineReader
}

func OpenFile(path string, maxLineBytes int) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	reader, err := newLineReader(file, maxLineBytes)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &File{file: file, path: path, reader: reader}, nil
}

func (s *File) Descriptor() logs.SourceDescriptor {
	return fileDescriptor(s.path)
}

func (s *File) Next(ctx context.Context) (logs.SourceLine, error) {
	return s.reader.next(ctx)
}

func (s *File) Close() error {
	if s == nil || s.file == nil {
		return nil
	}
	return s.file.Close()
}

// Follow waits for newline-terminated lines appended to path. It resets to
// the beginning when the file is truncated.
type Follow struct {
	path         string
	maxLineBytes int
	pollInterval time.Duration
	offset       int64
	lineNumber   int
	pending      []byte
}

func OpenFollow(path string, maxLineBytes int, pollInterval time.Duration) (*Follow, error) {
	if maxLineBytes <= 0 {
		return nil, fmt.Errorf("max line bytes must be positive: %d", maxLineBytes)
	}
	if pollInterval <= 0 {
		return nil, fmt.Errorf("poll interval must be positive: %s", pollInterval)
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("stat log file: %w", err)
	}
	return &Follow{path: path, maxLineBytes: maxLineBytes, pollInterval: pollInterval}, nil
}

func (s *Follow) Descriptor() logs.SourceDescriptor {
	return fileDescriptor(s.path)
}

func (s *Follow) Next(ctx context.Context) (logs.SourceLine, error) {
	for {
		if err := ctx.Err(); err != nil {
			return logs.SourceLine{}, err
		}

		line, found, err := s.readLine()
		if err != nil {
			return logs.SourceLine{}, err
		}
		if found {
			s.lineNumber++
			return logs.SourceLine{Number: s.lineNumber, Data: line}, nil
		}

		timer := time.NewTimer(s.pollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return logs.SourceLine{}, ctx.Err()
		case <-timer.C:
		}
	}
}

func (s *Follow) readLine() ([]byte, bool, error) {
	if index := newlineIndex(s.pending); index >= 0 {
		if index > s.maxLineBytes {
			return nil, false, ErrLineTooLong
		}
		line := append([]byte(nil), s.pending[:index]...)
		s.pending = append(s.pending[:0], s.pending[index+1:]...)
		return trimCarriageReturn(line), true, nil
	}

	info, err := os.Stat(s.path)
	if err != nil {
		return nil, false, fmt.Errorf("stat followed log file: %w", err)
	}
	if info.Size() < s.offset {
		s.offset = 0
		s.pending = nil
	}
	if info.Size() == s.offset {
		return nil, false, nil
	}

	file, err := os.Open(s.path)
	if err != nil {
		return nil, false, fmt.Errorf("open followed log file: %w", err)
	}
	defer file.Close()

	chunk := make([]byte, 32*1024)
	for s.offset < info.Size() {
		remaining := info.Size() - s.offset
		if int64(len(chunk)) > remaining {
			chunk = chunk[:remaining]
		}
		n, readErr := file.ReadAt(chunk, s.offset)
		if n > 0 {
			s.offset += int64(n)
			s.pending = append(s.pending, chunk[:n]...)
			if index := newlineIndex(s.pending); index >= 0 {
				if index > s.maxLineBytes {
					return nil, false, ErrLineTooLong
				}
				line := append([]byte(nil), s.pending[:index]...)
				s.pending = append(s.pending[:0], s.pending[index+1:]...)
				return trimCarriageReturn(line), true, nil
			}
			if len(s.pending) > s.maxLineBytes {
				return nil, false, ErrLineTooLong
			}
		}
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, false, fmt.Errorf("read followed log file: %w", readErr)
		}
		if n == 0 {
			break
		}
	}
	return nil, false, nil
}

type lineReader struct {
	reader       *bufio.Reader
	maxLineBytes int
	lineNumber   int
}

func newLineReader(input io.Reader, maxLineBytes int) (*lineReader, error) {
	if input == nil {
		return nil, errors.New("log input is required")
	}
	if maxLineBytes <= 0 {
		return nil, fmt.Errorf("max line bytes must be positive: %d", maxLineBytes)
	}
	return &lineReader{
		reader:       bufio.NewReaderSize(input, min(maxLineBytes+1, 32*1024)),
		maxLineBytes: maxLineBytes,
	}, nil
}

func (r *lineReader) next(ctx context.Context) (logs.SourceLine, error) {
	if err := ctx.Err(); err != nil {
		return logs.SourceLine{}, err
	}

	var line []byte
	for {
		fragment, err := r.reader.ReadSlice('\n')
		line = append(line, fragment...)
		if len(line) > r.maxLineBytes+1 {
			return logs.SourceLine{}, ErrLineTooLong
		}
		switch {
		case err == nil:
			line = line[:len(line)-1]
			if len(line) > r.maxLineBytes {
				return logs.SourceLine{}, ErrLineTooLong
			}
			r.lineNumber++
			return logs.SourceLine{Number: r.lineNumber, Data: trimCarriageReturn(line)}, nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			if len(line) == 0 {
				return logs.SourceLine{}, io.EOF
			}
			if len(line) > r.maxLineBytes {
				return logs.SourceLine{}, ErrLineTooLong
			}
			r.lineNumber++
			return logs.SourceLine{Number: r.lineNumber, Data: trimCarriageReturn(line)}, nil
		default:
			return logs.SourceLine{}, fmt.Errorf("read log input: %w", err)
		}
	}
}

func fileDescriptor(path string) logs.SourceDescriptor {
	return logs.SourceDescriptor{ID: path, Name: filepath.Base(path), Kind: "file"}
}

func newlineIndex(data []byte) int {
	for index, value := range data {
		if value == '\n' {
			return index
		}
	}
	return -1
}

func trimCarriageReturn(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[:len(data)-1]
	}
	return data
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
