package dashboard

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var ErrEntryNotFound = fmt.Errorf("entry not found")

type Store interface {
	List() ([]Entry, error)
	Get(id string) (Entry, error)
	Put(entry Entry) error
	Delete(id string) error
	SaveUpload(src io.Reader) (string, error)
	ResolveUpload(path string) string
}

type FileStore struct {
	metadataPath string
	uploadDir    string
	mutex        sync.Mutex
}

func NewFileStore(storePath, uploadDir string) (*FileStore, error) {
	if err := os.MkdirAll(storePath, 0o755); err != nil {
		return nil, fmt.Errorf("create store path: %w", err)
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload directory: %w", err)
	}
	return &FileStore{
		metadataPath: filepath.Join(storePath, "entries.json"),
		uploadDir:    uploadDir,
	}, nil
}

func (s *FileStore) List() ([]Entry, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	entries, err := s.readLocked()
	if err != nil {
		return nil, err
	}
	out := make([]Entry, len(entries))
	copy(out, entries)
	return out, nil
}

func (s *FileStore) Get(id string) (Entry, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	entries, err := s.readLocked()
	if err != nil {
		return Entry{}, err
	}
	for _, entry := range entries {
		if entry.ID == id {
			return entry, nil
		}
	}
	return Entry{}, ErrEntryNotFound
}

func (s *FileStore) Put(entry Entry) error {
	if err := entry.Validate(); err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	entries, err := s.readLocked()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	updated := false
	var staleUploadPath string
	for i := range entries {
		if entries[i].ID != entry.ID {
			continue
		}
		if entries[i].UploadedTemplatePath != "" && entries[i].UploadedTemplatePath != entry.UploadedTemplatePath {
			staleUploadPath = entries[i].UploadedTemplatePath
		}
		entry.CreatedAt = entries[i].CreatedAt
		entry.UpdatedAt = now
		entries[i] = entry
		updated = true
		break
	}
	if !updated {
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = now
		}
		entry.UpdatedAt = now
		entries = append(entries, entry)
	}

	if err := s.writeLocked(entries); err != nil {
		return err
	}

	if staleUploadPath != "" {
		if err := os.Remove(s.ResolveUpload(staleUploadPath)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove replaced upload: %w", err)
		}
	}

	return nil
}

func (s *FileStore) Delete(id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	entries, err := s.readLocked()
	if err != nil {
		return err
	}

	index := -1
	var uploadPath string
	for i := range entries {
		if entries[i].ID == id {
			index = i
			uploadPath = entries[i].UploadedTemplatePath
			break
		}
	}
	if index == -1 {
		return nil
	}

	entries = append(entries[:index], entries[index+1:]...)
	if err := s.writeLocked(entries); err != nil {
		return err
	}

	if uploadPath != "" {
		if err := os.Remove(s.ResolveUpload(uploadPath)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove uploaded file: %w", err)
		}
	}

	return nil
}

func (s *FileStore) SaveUpload(src io.Reader) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate upload filename: %w", err)
	}
	filename := hex.EncodeToString(random) + ".rdp"
	fullPath := filepath.Join(s.uploadDir, filename)

	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create upload file: %w", err)
	}

	if _, err := io.Copy(file, src); err != nil {
		_ = file.Close()
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("write upload file: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(fullPath)
		return "", fmt.Errorf("close upload file: %w", err)
	}

	return filename, nil
}

func (s *FileStore) ResolveUpload(path string) string {
	return filepath.Join(s.uploadDir, filepath.Base(path))
}

func (s *FileStore) readLocked() ([]Entry, error) {
	data, err := os.ReadFile(s.metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("read metadata: %w", err)
	}
	if len(data) == 0 {
		return []Entry{}, nil
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	if entries == nil {
		return []Entry{}, nil
	}
	return entries, nil
}

func (s *FileStore) writeLocked(entries []Entry) error {
	if err := os.MkdirAll(filepath.Dir(s.metadataPath), 0o755); err != nil {
		return fmt.Errorf("ensure metadata directory: %w", err)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(s.metadataPath), "entries-*.tmp")
	if err != nil {
		return fmt.Errorf("create metadata temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write metadata temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close metadata temp file: %w", err)
	}

	if err := os.Rename(tmpName, s.metadataPath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("replace metadata file: %w", err)
	}
	return nil
}
