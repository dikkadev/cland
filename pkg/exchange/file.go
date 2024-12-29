package exchange

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Handler struct {
	InputDir  string
	ErrorDir  string
	Running   bool
	Processes *sync.Pool
}

func NewHandler(inputDir, errorDir string) *Handler {
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		slog.Info("Creating input directory", "dir", inputDir)
		err = os.MkdirAll(inputDir, 0755)
		if err != nil {
			panic(err)
		}
	}
	if _, err := os.Stat(errorDir); os.IsNotExist(err) {
		slog.Info("Creating error directory", "dir", errorDir)
		err = os.MkdirAll(errorDir, 0755)
		if err != nil {
			panic(err)
		}
	}
	return &Handler{
		InputDir: inputDir,
		ErrorDir: errorDir,
		Running:  false,
		Processes: &sync.Pool{
			New: func() any {
				return &Process{}
			},
		},
	}
}

func (h *Handler) Start() error {
	slog.Info("Starting handler", "input", h.InputDir, "error", h.ErrorDir)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Error creating watcher", "err", err)
		return err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create {
					p := h.Processes.Get().(*Process)
					p.Filepath = event.Name

					go func(proc *Process) {
						defer func() {
							proc.Filepath = ""
							proc.Notif = nil
							h.Processes.Put(proc)
						}()

						slog.Info("New file created", "file", proc.Filepath)
						err := proc.ReadFile()
						if err != nil {
							slog.Error("Error reading file", "err", err)
							err = h.errorFile(proc)
							if err != nil {
								slog.Error("Error moving file to error dir", "err", err)
							}
							return
						}

						slog.Info("Notification parsed", "topic", proc.Notif.Topic, "metadata", proc.Notif.Metadata, "message", proc.Notif.Message)
					}(p)
				}
			case werr := <-watcher.Errors:
				slog.Error("Watcher error", "err", werr)
			}
		}
	}()

	return watcher.Add(h.InputDir)
}

func (h *Handler) errorFile(p *Process) error {
	filename := filepath.Base(p.Filepath)
	errorPath := filepath.Join(h.ErrorDir, filename)

	if _, err := os.Stat(errorPath); err == nil {
		timestamp := time.Now().Format("20060102150405")
		errorPath = filepath.Join(h.ErrorDir, fmt.Sprintf("%s_%s", filename, timestamp))
	}

	return os.Rename(p.Filepath, errorPath)
}

type Process struct {
	Filepath string
	Notif    *Notification
}

const (
	READ_FILE_MAX_ATTEMPTS = 5
	READ_FILE_RETRY_DELAY  = 200 * time.Millisecond
)

func (p *Process) ReadFile() error {
	var content []byte
	var err error
	for attempt := 1; attempt <= READ_FILE_MAX_ATTEMPTS; attempt++ {
		content, err = os.ReadFile(p.Filepath)
		if err != nil {
			slog.Warn("Failed to read file, retrying", "attempt", attempt, "err", err)
			time.Sleep(READ_FILE_RETRY_DELAY)
			continue
		}
		if len(content) == 0 {
			slog.Warn("File is empty, retrying", "attempt", attempt)
			time.Sleep(READ_FILE_RETRY_DELAY)
			continue
		}
		break
	}
	if err != nil {
		return err
	}
	if len(content) == 0 {
		return errors.New("file content is empty after retries")
	}

	lines := strings.Split(string(content), "\n")
	notif, err := parse(lines)
	if err != nil {
		return err
	}

	p.Notif = notif
	return nil
}

func parse(lines []string) (*Notification, error) {
	head := make([]string, 0)
	message := make([]string, 0)
	insideHead := true
	for _, line := range lines {
		if isRule(line) {
			insideHead = false
			continue
		}
		if insideHead {
			head = append(head, line)
		} else {
			message = append(message, line)
		}
	}
	slog.Debug("Parsed file", "head", head, "message", message)

	head = cleanHead(head)
	if len(head) < 1 {
		return nil, &NoTopicError{}
	}

	if len(message) < 1 {
		return nil, &EmptyMessageError{}
	}

	return &Notification{
		Topic:    head[0],
		Metadata: parseMetadata(head[1:]),
		Message:  strings.Join(message, "\n"),
	}, nil
}

func cleanHead(head []string) []string {
	cleaned := make([]string, 0)
	for _, line := range head {
		if line == "" || isComment(line) {
			continue
		}
		cleaned = append(cleaned, line)
	}
	return cleaned
}

func isRule(line string) bool {
	return strings.HasPrefix(line, "---")
}

func isComment(line string) bool {
	return strings.HasPrefix(line, "--")
}

func parseMetadata(lines []string) map[string]string {
	metadata := make(map[string]string)
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		metadata[key] = value
	}
	return metadata
}
