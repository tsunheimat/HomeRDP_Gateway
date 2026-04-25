package dashboard

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
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
	options      FileStoreOptions
	mutex        sync.Mutex
}

type FileStoreOptions struct {
	MaxUploads      int
	MaxStorageBytes int64
}

func NewFileStore(storePath, uploadDir string, options ...FileStoreOptions) (*FileStore, error) {
	if err := os.MkdirAll(storePath, 0o755); err != nil {
		return nil, fmt.Errorf("create store path: %w", err)
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload directory: %w", err)
	}
	opts := FileStoreOptions{}
	if len(options) > 0 {
		opts = options[0]
	}
	return &FileStore{
		metadataPath: filepath.Join(storePath, "entries.json"),
		uploadDir:    uploadDir,
		options:      opts,
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
	entry.Icon = NormalizeEntryIcon(entry.Icon)

	if err := entry.Validate(); err != nil {
		return err
	}

	if entry.Type == EntryTypeTemplate {
		if _, err := os.Stat(s.ResolveUpload(entry.UploadedTemplatePath)); err != nil {
			if os.IsNotExist(err) {
				return validationError("uploaded template does not exist")
			}
			return fmt.Errorf("stat uploaded template: %w", err)
		}
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
		_ = removeUploadFile(s.ResolveUpload(staleUploadPath))
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
		_ = removeUploadFile(s.ResolveUpload(uploadPath))
	}

	return nil
}

func (s *FileStore) SaveUpload(src io.Reader) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, err := io.ReadAll(src)
	if err != nil {
		return "", fmt.Errorf("read upload: %w", err)
	}
	if err := enforceFileQuota(s.uploadDir, s.options.MaxUploads, s.options.MaxStorageBytes, int64(len(data)), "template upload"); err != nil {
		return "", err
	}

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

	if _, err := file.Write(data); err != nil {
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

func removeUploadFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func enforceFileQuota(dir string, maxFiles int, maxStorageBytes int64, newFileBytes int64, label string) error {
	count, size, err := regularFileUsage(dir)
	if err != nil {
		return err
	}
	if maxFiles > 0 && count >= maxFiles {
		return validationError(fmt.Sprintf("%s count quota exceeded", label))
	}
	if maxStorageBytes > 0 {
		if size > maxStorageBytes || newFileBytes > maxStorageBytes-size {
			return validationError(fmt.Sprintf("%s storage quota exceeded", label))
		}
	}
	return nil
}

func regularFileUsage(dir string) (int, int64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, fmt.Errorf("read upload directory: %w", err)
	}

	var count int
	var size int64
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return 0, 0, fmt.Errorf("stat upload file: %w", err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		count++
		if info.Size() > math.MaxInt64-size {
			return 0, 0, fmt.Errorf("upload directory size overflow")
		}
		size += info.Size()
	}
	return count, size, nil
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
	for i := range entries {
		entries[i].Icon = NormalizeEntryIcon(entries[i].Icon)
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
