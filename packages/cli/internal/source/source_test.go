// Nome: source_test.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Testa os adaptadores de fonte da CLI, garantindo leitura correta de
// stdin, arquivos acompanhados em modo follow, linhas incompletas, cancelamento
// por contexto e rejeição de entradas acima do limite configurado.
package source

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStdinEmitsFinalLineWithoutNewline(t *testing.T) {
	t.Parallel()

	source, err := NewStdin(strings.NewReader("first\nlast"), 1024)
	if err != nil {
		t.Fatalf("NewStdin() error = %v", err)
	}

	first, err := source.Next(context.Background())
	if err != nil || first.Number != 1 || string(first.Data) != "first" {
		t.Fatalf("first line = %#v, %v", first, err)
	}
	last, err := source.Next(context.Background())
	if err != nil || last.Number != 2 || string(last.Data) != "last" {
		t.Fatalf("last line = %#v, %v", last, err)
	}
	if _, err := source.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("final error = %v, want EOF", err)
	}
}

func TestStdinRejectsLineOverLimit(t *testing.T) {
	t.Parallel()

	source, err := NewStdin(strings.NewReader("12345\n"), 4)
	if err != nil {
		t.Fatalf("NewStdin() error = %v", err)
	}
	if _, err := source.Next(context.Background()); !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("Next() error = %v, want ErrLineTooLong", err)
	}
}

func TestFollowWaitsForCompleteLineAndHonorsCancellation(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "application.ndjson")
	if err := os.WriteFile(path, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	follow, err := OpenFollow(path, 1024, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("OpenFollow() error = %v", err)
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		file, openErr := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
		if openErr != nil {
			t.Errorf("open append file: %v", openErr)
			return
		}
		defer file.Close()
		if _, writeErr := file.WriteString(" line\nsecond\n"); writeErr != nil {
			t.Errorf("append log lines: %v", writeErr)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	first, err := follow.Next(ctx)
	if err != nil || first.Number != 1 || string(first.Data) != "partial line" {
		t.Fatalf("first line = %#v, %v", first, err)
	}
	second, err := follow.Next(ctx)
	if err != nil || second.Number != 2 || string(second.Data) != "second" {
		t.Fatalf("second line = %#v, %v", second, err)
	}

	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := follow.Next(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("Next() error = %v, want context cancellation", err)
	}
}

func TestFollowReadsSeveralShortLinesFromOneChunk(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "application.ndjson")
	if err := os.WriteFile(path, []byte("a\na\na\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	follow, err := OpenFollow(path, 1, time.Millisecond)
	if err != nil {
		t.Fatalf("OpenFollow() error = %v", err)
	}

	for number := 1; number <= 3; number++ {
		line, nextErr := follow.Next(context.Background())
		if nextErr != nil || line.Number != number || string(line.Data) != "a" {
			t.Fatalf("line %d = %#v, %v", number, line, nextErr)
		}
	}
}

func TestFollowNormalizesCRLFForBufferedLines(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "application.ndjson")
	if err := os.WriteFile(path, []byte("first\r\nsecond\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	follow, err := OpenFollow(path, 1024, time.Millisecond)
	if err != nil {
		t.Fatalf("OpenFollow() error = %v", err)
	}

	for number, want := range []string{"first", "second"} {
		line, nextErr := follow.Next(context.Background())
		if nextErr != nil || line.Number != number+1 || string(line.Data) != want {
			t.Fatalf("line %d = %#v, %v", number+1, line, nextErr)
		}
	}
}
