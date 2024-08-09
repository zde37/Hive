package ipfs

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/stretchr/testify/require"
)

var (
	testClient Client
	peerId     = "12D3KooWAd56wFkoAB6weMT9KY6XJ5FWZdDkxCi1QpWHe4ySD1kV" // replace with your peerId when testing
)

func TestMain(m *testing.M) {
	rpcAddr := "/ip4/127.0.0.1/tcp/5001"
	rpc, err := NewClient(rpcAddr)
	if err != nil {
		log.Fatalf("Failed to create IPFS client: %v", err)
	}
	testClient = NewClientImpl(rpc)

	os.Exit(m.Run())
}

func SubTestNodeInfo(peerID string, t *testing.T) {
	tests := []struct {
		name    string
		peerID  string
		wantErr bool
	}{
		{
			name:    "Valid peer ID",
			peerID:  peerID,
			wantErr: false,
		},
		{
			name:    "Empty peer ID",
			peerID:  "",
			wantErr: true,
		},
		{
			name:    "Invalid peer ID format",
			peerID:  "invalid-peer-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := testClient.NodeInfo(ctx, tt.peerID)
			require.Equal(t, tt.wantErr, err != nil)

			// if tt.wantErr {
			// 	if err == nil {
			// 		t.Errorf("Expected error, but got nil")
			// 	}
			// } else {
			// 	if err != nil {
			// 		t.Errorf("Unexpected error: %v", err)
			// 	}
			// 	if nodeInfo.ID == "" {
			// 		t.Errorf("Expected non-empty node ID, but got empty")
			// 	}
			// 	if len(nodeInfo.Addresses) == 0 {
			// 		t.Errorf("Expected non-empty addresses, but got empty")
			// 	}
			// }
		})
	}
}

func TestAdd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ipfs-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		fileName string
		filePath string
		setup    func() error
		wantErr  bool
	}{
		{
			name:     "Add single file",
			fileName: "test.txt",
			filePath: filepath.Join(tempDir, "test.txt"),
			setup: func() error {
				return os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test content"), 0644)
			},
			wantErr: false,
		},
		{
			name:     "Add directory",
			fileName: "testdir",
			filePath: filepath.Join(tempDir, "testdir"),
			setup: func() error {
				return os.Mkdir(filepath.Join(tempDir, "testdir"), 0755)
			},
			wantErr: false,
		},
		{
			name:     "Empty directory name",
			fileName: "",
			filePath: filepath.Join(tempDir, "testdir"),
			setup:    func() error { return nil },
			wantErr:  true,
		},
		{
			name:     "Empty file name",
			fileName: "",
			filePath: filepath.Join(tempDir, "test.txt"),
			setup:    func() error { return nil },
			wantErr:  true,
		},
		{ 
			name:     "Empty file path",
			fileName: "test.txt",
			filePath: "",
			setup:    func() error { return nil },
			wantErr:  true,
		},
		{
			name:     "Empty directory path",
			fileName: "testdir",
			filePath: "",
			setup:    func() error { return nil },
			wantErr:  true,
		},
		{
			name:     "Non-existent file",
			fileName: "nonexistent.txt",
			filePath: filepath.Join(tempDir, "nonexistent.txt"),
			setup:    func() error { return nil },
			wantErr:  true,
		},
		{
			name:     "Non-existent directory",
			fileName: "nonexistentdir",
			filePath: filepath.Join(tempDir, "nonexistentdir"),
			setup:    func() error { return nil },
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.setup())

			ctx := context.Background()
			path, cid, err := testClient.Add(ctx, tt.fileName, tt.filePath)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, path)
				require.Empty(t, cid)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, path)
				require.NotEmpty(t, cid) 
				delete(ctx, path, t)
			}
		})
	}
}

func addFile(ctx context.Context, t *testing.T) (string, string) {
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("ipfs-test-%s", time.Now().String()))
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fileName := "test.txt"
	filePath := filepath.Join(tempDir, fileName)
	err = os.WriteFile(filepath.Join(tempDir, fileName), []byte("test content"), 0644)
	require.NoError(t, err)

	path, cid, err := testClient.Add(ctx, fileName, filePath)
	require.NoError(t, err)
	require.NotEmpty(t, cid)
	require.NotEmpty(t, path)
	return path, cid
}

func addFolder(ctx context.Context, t *testing.T) (string, string) {
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("ipfs-test-%s", time.Now().String()))
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dirName := "testdir"
	dirPath := filepath.Join(tempDir, dirName)
	err = os.Mkdir(filepath.Join(tempDir, dirName), 0755)
	require.NoError(t, err)

	// Add a file to the folder
	fileName := "testfile.txt"
	filePath := filepath.Join(dirPath, fileName)
	err = os.WriteFile(filePath, []byte("Test content in folder"), 0644)
	require.NoError(t, err)

	path, cid, err := testClient.Add(ctx, dirName, dirPath)
	require.NoError(t, err)
	require.NotEmpty(t, cid)
	require.NotEmpty(t, path)
	return path, cid
}

func delete(ctx context.Context, path string, t *testing.T) {
	err := testClient.DeleteFile(ctx, path)
	require.NoError(t, err)
}

func TestDisplayFileContent(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		setup    func(context.Context, *testing.T) (string, string)
		wantErr  bool
		cleanup  func(context.Context, string, *testing.T)
	}{
		{
			name:     "Display content of existing file",
			filePath: "",
			setup:    addFile,
			wantErr:  false,
			cleanup:  delete,
		},
		{
			name:     "Attempt to display content of a directory",
			filePath: "",
			setup:    addFolder,
			wantErr:  true,
			cleanup:  delete,
		},
		{
			name:     "Attempt to display content of non-existent file",
			filePath: "/non/existent/file.txt",
			setup:    func(context.Context, *testing.T) (string, string) { return "", "" },
			wantErr:  true,
			cleanup:  func(context.Context, string, *testing.T) {},
		},
		{
			name:     "Empty file path",
			filePath: "",
			setup:    func(context.Context, *testing.T) (string, string) { return "", "" },
			wantErr:  true,
			cleanup:  func(context.Context, string, *testing.T) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			path, _ := tt.setup(ctx, t)
			if tt.filePath == "" && path != "" {
				tt.filePath = path
			}

			content, err := testClient.DisplayFileContent(ctx, tt.filePath)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, content)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, content)
				require.Contains(t, content, "test content")
			}

			if tt.cleanup != nil {
				tt.cleanup(ctx, path, t)
			}
		})
	}
}

func TestDisplayFileContentLargeFile(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "ipfs-test-large-file")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fileName := "large_file.txt"
	filePath := filepath.Join(tempDir, fileName)

	// Create a large file (10 MB)
	f, err := os.Create(filePath)
	require.NoError(t, err)
	defer f.Close()

	size := 10 * 1024 * 1024 // 10 MB
	_, err = f.WriteString(strings.Repeat("a", size))
	require.NoError(t, err)

	path, cid, err := testClient.Add(ctx, fileName, filePath)
	require.NoError(t, err)
	require.NotEmpty(t, cid)
	require.NotEmpty(t, path)

	content, err := testClient.DisplayFileContent(ctx, path)
	require.NoError(t, err)
	require.NotEmpty(t, content)
	require.Equal(t, size, len(content))

	err = testClient.DeleteFile(ctx, path)
	require.NoError(t, err)
}

func TestDisplayFileContentConcurrent(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "ipfs-test-concurrent")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	numFiles := 5
	var wg sync.WaitGroup
	wg.Add(numFiles)

	for i := 0; i < numFiles; i++ {
		go func(index int) {
			defer wg.Done()
			fileName := fmt.Sprintf("file_%d.txt", index)
			filePath := filepath.Join(tempDir, fileName)
			err := os.WriteFile(filePath, []byte(fmt.Sprintf("content of file %d", index)), 0644)
			require.NoError(t, err)

			path, _, err := testClient.Add(ctx, fileName, filePath)
			require.NoError(t, err)

			content, err := testClient.DisplayFileContent(ctx, path)
			require.NoError(t, err)
			require.Contains(t, content, fmt.Sprintf("content of file %d", index))

			err = testClient.DeleteFile(ctx, path)
			require.NoError(t, err)
		}(i)
	}

	wg.Wait()
}

func TestPinObject(t *testing.T) {
	ctx := context.Background()
	path, cid := addFile(ctx, t)
	require.NotEmpty(t, cid)
	require.NotEmpty(t, cid)
	defer delete(ctx, path, t)

	tests := []struct {
		name       string
		objectName string
		objectPath string
		wantErr    bool
	}{
		{
			name:       "Valid pin",
			objectName: "test-pin",
			objectPath: path,
			wantErr:    false,
		},
		{
			name:       "Empty object name",
			objectName: "",
			objectPath: path,
			wantErr:    false,
		},
		{
			name:       "Invalid object path",
			objectName: "test-pin",
			objectPath: "invalid-path",
			wantErr:    true,
		},
		{
			name:       "Empty object path",
			objectName: "test-pin",
			objectPath: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := testClient.PinObject(ctx, tt.objectName, tt.objectPath)
			require.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestPinObjectConcurrent(t *testing.T) {
	ctx := context.Background()
	numPins := 5
	var wg sync.WaitGroup
	wg.Add(numPins)

	path, cid := addFile(ctx, t)
	require.NotEmpty(t, cid)
	require.NotEmpty(t, cid)
	defer delete(ctx, path, t)

	for i := 0; i < numPins; i++ {
		go func(index int) {
			defer wg.Done()

			objectName := fmt.Sprintf("test-pin-%d", index)
			err := testClient.PinObject(ctx, objectName, path)
			require.NoError(t, err)
		}(i)
	}

	wg.Wait()
}

func TestDeleteFile(t *testing.T) {
	ctx := context.Background()
	path, cid := addFile(ctx, t)
	require.NotEmpty(t, cid)
	require.NotEmpty(t, cid)
	delete(ctx, path, t)
}

func TestDownloadFile(t *testing.T) {
	tests := []struct {
		name    string
		cid     string
		setup   func(context.Context, *testing.T) (string, string)
		wantErr bool
	}{
		{
			name:    "Download existing file",
			cid:     "",
			setup:   addFile,
			wantErr: false,
		},
		{
			name:    "Download non-existent file",
			cid:     "QmInvalidCID",
			setup:   func(context.Context, *testing.T) (string, string) { return "", "" },
			wantErr: true,
		},
		{
			name:    "Empty CID",
			cid:     "",
			setup:   func(context.Context, *testing.T) (string, string) { return "", "" },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			path, cid := tt.setup(ctx, t)
			if tt.cid == "" && cid != "" {
				tt.cid = cid
			}

			content, err := testClient.DownloadFile(ctx, tt.cid)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, content)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, content)
				require.Equal(t, "test content", string(content))
				delete(ctx, path, t)
			}
		})
	}
}

func TestDownloadFileLarge(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "ipfs-test-large-download")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fileName := "large_download.txt"
	filePath := filepath.Join(tempDir, fileName)

	size := 10 * 1024 * 1024 // 10 MB
	content := strings.Repeat("a", size)

	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	path, cid, err := testClient.Add(ctx, fileName, filePath)
	require.NoError(t, err)
	require.NotEmpty(t, cid)

	downloadedContent, err := testClient.DownloadFile(ctx, cid)
	require.NoError(t, err)
	require.NotEmpty(t, downloadedContent)
	require.Equal(t, size, len(downloadedContent))
	require.Equal(t, content, string(downloadedContent))
	delete(ctx, path, t)
}

func TestDownloadFileConcurrent(t *testing.T) {
	ctx := context.Background()
	numFiles := 5
	var wg sync.WaitGroup
	wg.Add(numFiles)

	for i := 0; i < numFiles; i++ {
		go func(index int) {
			defer wg.Done()
			tempDir, err := os.MkdirTemp("", fmt.Sprintf("ipfs-test-concurrent-download-%d", index))
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			fileName := fmt.Sprintf("file_%d.txt", index)
			filePath := filepath.Join(tempDir, fileName)
			content := fmt.Sprintf("content of file %d", index)
			err = os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)

			path, cid, err := testClient.Add(ctx, fileName, filePath)
			require.NoError(t, err)
			require.NotEmpty(t, cid)
			defer delete(ctx, path, t)

			downloadedContent, err := testClient.DownloadFile(ctx, cid)
			require.NoError(t, err)
			require.NotEmpty(t, downloadedContent)
			require.Equal(t, content, string(downloadedContent))
		}(i)
	}

	wg.Wait()
}

func TestDownloadDir(t *testing.T) {
	tests := []struct {
		name       string
		cid        string
		outputPath string
		setup      func(context.Context, *testing.T) (string, string)
		wantErr    bool
	}{
		{
			name:       "Download existing directory",
			cid:        "",
			outputPath: "",
			setup:      addFolder,
			wantErr:    false,
		},
		{
			name:       "Download non-existent directory",
			cid:        "QmInvalidDirCID",
			outputPath: "",
			setup:      func(context.Context, *testing.T) (string, string) { return "", "" },
			wantErr:    true,
		},
		{
			name:       "Empty CID",
			cid:        "",
			outputPath: "",
			setup:      func(context.Context, *testing.T) (string, string) { return "", "" },
			wantErr:    true,
		},
		{
			name:       "Invalid output path",
			cid:        "",
			outputPath: "/nonexistent/path",
			setup:      addFolder,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			path, cid := tt.setup(ctx, t)
			if tt.cid == "" && cid != "" {
				tt.cid = cid
			}

			tempDir, err := os.MkdirTemp("", "ipfs-test-download-dir")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)
			if path != "" {
				defer delete(ctx, path, t)
			}

			if tt.outputPath == "" {
				tt.outputPath = tempDir
			}

			err = testClient.DownloadDir(ctx, tt.cid, tt.outputPath)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.DirExists(t, tt.outputPath)
				files, err := os.ReadDir(tt.outputPath)
				require.NoError(t, err)
				require.NotEmpty(t, files)
			}
		})
	}
}

func TestDownloadDirLarge(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "ipfs-test-large-download-dir")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dirName := "large_dir"
	dirPath := filepath.Join(tempDir, dirName)
	err = os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	numFiles := 100
	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("file_%d.txt", i)
		filePath := filepath.Join(dirPath, fileName)
		content := strings.Repeat(fmt.Sprintf("content of file %d", i), 1024) // ~10KB per file
		err = os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	path, cid, err := testClient.Add(ctx, dirName, dirPath)
	require.NoError(t, err)
	require.NotEmpty(t, cid)

	outputDir, err := os.MkdirTemp("", "ipfs-test-large-download-output")
	require.NoError(t, err)
	defer os.RemoveAll(outputDir)

	err = testClient.DownloadDir(ctx, cid, outputDir)
	require.NoError(t, err)

	downloadedFiles, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	require.Len(t, downloadedFiles, numFiles)

	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("file_%d.txt", i)
		filePath := filepath.Join(outputDir, fileName)
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		require.Contains(t, string(content), fmt.Sprintf("content of file %d", i))
	}

	delete(ctx, path, t)
}

func TestDownloadDirConcurrent(t *testing.T) {
	ctx := context.Background()
	numDirs := 5
	var wg sync.WaitGroup
	wg.Add(numDirs)

	for i := 0; i < numDirs; i++ {
		go func(index int) {
			defer wg.Done()
			tempDir, err := os.MkdirTemp("", fmt.Sprintf("ipfs-test-concurrent-download-dir-%d", index))
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			dirName := fmt.Sprintf("dir_%d", index)
			dirPath := filepath.Join(tempDir, dirName)
			err = os.Mkdir(dirPath, 0755)
			require.NoError(t, err)

			fileName := fmt.Sprintf("file_%d.txt", index)
			filePath := filepath.Join(dirPath, fileName)
			content := fmt.Sprintf("content of file %d", index)
			err = os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)

			path, cid, err := testClient.Add(ctx, dirName, dirPath)
			require.NoError(t, err)
			require.NotEmpty(t, cid)
			defer delete(ctx, path, t)

			outputDir, err := os.MkdirTemp("", fmt.Sprintf("ipfs-test-concurrent-download-output-%d", index))
			require.NoError(t, err)
			defer os.RemoveAll(outputDir)

			err = testClient.DownloadDir(ctx, cid, outputDir)
			require.NoError(t, err)

			downloadedFiles, err := os.ReadDir(outputDir)
			require.NoError(t, err)
			require.Len(t, downloadedFiles, 1)

			downloadedContent, err := os.ReadFile(filepath.Join(outputDir, fileName))
			require.NoError(t, err)
			require.Equal(t, content, string(downloadedContent))
		}(i)
	}

	wg.Wait()
}

func TestListPins(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(context.Context, *testing.T) (string, string)
		wantErr bool
	}{
		{
			name:    "List pins with existing pinned objects",
			setup:   addFile,
			wantErr: false,
		},
		{
			name:    "List pins with no pinned objects",
			setup:   func(context.Context, *testing.T) (string, string) { return "", "" },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			path, _ := tt.setup(ctx, t)
			if path != "" {
				defer delete(ctx, path, t)
			}

			pins, err := testClient.ListPins(ctx)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, pins)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pins)

				pinsMap, ok := pins.(map[string]interface{})
				require.True(t, ok, "Expected pins to be a map[string]interface{}")

				keys, ok := pinsMap["Keys"].(map[string]interface{})
				log.Printf("%v", pinsMap)
				log.Printf("%v", pinsMap["Keys"])
				// require.True(t, ok, "Expected Keys to be a map[string]interface{}")

				if path != "" {
					require.NotEmpty(t, keys, "Expected at least one pinned object")
				}
			}
		})
	}
}

func TestListPinsConcurrent(t *testing.T) {
	ctx := context.Background()
	numConcurrent := 5
	var wg sync.WaitGroup
	wg.Add(numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func() {
			defer wg.Done()
			pins, err := testClient.ListPins(ctx)
			require.NoError(t, err)
			require.NotNil(t, pins)
		}()
	}

	wg.Wait()
}

func TestListPinsWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	pins, err := testClient.ListPins(ctx)
	require.Error(t, err)
	require.Nil(t, pins)
	require.Contains(t, err.Error(), "context canceled")
}

func TestListPinsWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond)

	pins, err := testClient.ListPins(ctx)
	require.Error(t, err)
	require.Nil(t, pins)
	require.Contains(t, err.Error(), "context deadline exceeded")
}

func TestListDir(t *testing.T) {
	tests := []struct {
		name    string
		dirPath string
		setup   func(context.Context, *testing.T) (string, string)
		wantErr bool
		want    int
	}{
		{
			name:    "List existing directory",
			dirPath: "",
			setup:   addFolder,
			wantErr: false,
			want:    1,
		},
		{
			name:    "List non-existent directory",
			dirPath: "/non/existent/dir",
			setup:   func(context.Context, *testing.T) (string, string) { return "", "" },
			wantErr: true,
			want:    0,
		},
		{
			name:    "Empty directory path",
			dirPath: "",
			setup:   func(context.Context, *testing.T) (string, string) { return "", "" },
			wantErr: true,
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			path, _ := tt.setup(ctx, t)
			if tt.dirPath == "" && path != "" {
				tt.dirPath = path
			}
			if path != "" {
				defer delete(ctx, path, t)
			}

			files, err := testClient.ListDir(ctx, tt.dirPath)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, files)
			} else {
				require.NoError(t, err)
				require.NotNil(t, files)
				require.Len(t, files, tt.want)
				for _, file := range files {
					require.NotEmpty(t, file.Name)
					require.NotEmpty(t, file.Cid)
					require.NotZero(t, file.Size)
					require.NotEmpty(t, file.Type)
				}
			}
		})
	}
}

func TestListDirLarge(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "ipfs-test-large-dir")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dirName := "large_dir"
	dirPath := filepath.Join(tempDir, dirName)
	err = os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	numFiles := 100
	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("file_%d.txt", i)
		filePath := filepath.Join(dirPath, fileName)
		content := fmt.Sprintf("content of file %d", i)
		err = os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	path, _, err := testClient.Add(ctx, dirName, dirPath)
	require.NoError(t, err)
	defer delete(ctx, path, t)

	files, err := testClient.ListDir(ctx, path)
	require.NoError(t, err)
	require.Len(t, files, numFiles)

	for _, file := range files {
		require.NotEmpty(t, file.Name)
		require.NotEmpty(t, file.Cid)
		require.NotZero(t, file.Size)
		require.Equal(t, iface.FileType(1), file.Type)
	}
}

func TestListDirConcurrent(t *testing.T) {
	ctx := context.Background()
	numDirs := 5
	var wg sync.WaitGroup
	wg.Add(numDirs)

	for i := 0; i < numDirs; i++ {
		go func(index int) {
			defer wg.Done()
			tempDir, err := os.MkdirTemp("", fmt.Sprintf("ipfs-test-concurrent-dir-%d", index))
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			dirName := fmt.Sprintf("dir_%d", index)
			dirPath := filepath.Join(tempDir, dirName)
			err = os.Mkdir(dirPath, 0755)
			require.NoError(t, err)

			numFiles := 5
			for j := 0; j < numFiles; j++ {
				fileName := fmt.Sprintf("file_%d.txt", j)
				filePath := filepath.Join(dirPath, fileName)
				content := fmt.Sprintf("content of file %d in dir %d", j, index)
				err = os.WriteFile(filePath, []byte(content), 0644)
				require.NoError(t, err)
			}

			path, _, err := testClient.Add(ctx, dirName, dirPath)
			require.NoError(t, err)
			defer delete(ctx, path, t)

			files, err := testClient.ListDir(ctx, path)
			require.NoError(t, err)
			require.Len(t, files, numFiles)

			for _, file := range files {
				require.NotEmpty(t, file.Name)
				require.NotEmpty(t, file.Cid)
				require.NotZero(t, file.Size)
				require.Equal(t, iface.FileType(1), file.Type)
			}
		}(i)
	}

	wg.Wait()
}

func TestListDirWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	files, err := testClient.ListDir(ctx, "/ipfs/bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354") // dummy folder cid
	require.Error(t, err)
	require.Nil(t, files)
	require.Contains(t, err.Error(), "context canceled")
}

func TestListDirWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond)

	files, err := testClient.ListDir(ctx, "/ipfs/bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354") // dummy folder cid
	require.Error(t, err)
	require.Nil(t, files)
	require.Contains(t, err.Error(), "context deadline exceeded")
}

func TestPing(t *testing.T) {
	tests := []struct {
		name    string
		peerID  string
		wantErr bool
	}{
		// {
		// 	name:    "Valid peer ID",
		// 	peerID:  "12D3KooWEEYvcSMGjVyENaUXPkpp7TQWhSbZipnpjPhBEXJVDCZ9", // use a valid  peer ID for this to work
		// 	wantErr: false,
		// },
		{
			name:    "Empty peer ID",
			peerID:  "",
			wantErr: true,
		},
		{
			name:    "Invalid peer ID format",
			peerID:  "invalid-peer-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			pingInfo, err := testClient.Ping(ctx, tt.peerID)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, pingInfo)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pingInfo)
				require.NotEmpty(t, pingInfo)
				for _, pi := range pingInfo {
					require.NotEmpty(t, pi.Success)
					require.NotZero(t, pi.Time)
					require.NotEmpty(t, pi.Text)
				}
			}
		})
	}
}

func TestNodeInfo(t *testing.T) {
	tests := []struct {
		name    string
		peerID  string
		wantErr bool
	}{
		// {
		// 	name:    "Valid peer ID", // use an actual valid peerid for this
		// 	peerID:  peerId,
		// 	wantErr: false,
		// },
		{
			name:    "Very long peer ID",
			peerID:  strings.Repeat("Q", 1000),
			wantErr: true,
		},
		{
			name:    "Peer ID with special characters",
			peerID:  "Qm!@#$%^&*()",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			info, err := testClient.NodeInfo(ctx, tt.peerID)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, info)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, info.ID)
				require.NotEmpty(t, info.Addresses)
				require.NotEmpty(t, info.Protocols)
				require.NotEmpty(t, info.PublicKey)
				require.NotEmpty(t, info.AgentVersion)
			}
		})
	}
}

func TestNodeInfoWithContext(t *testing.T) {
	peerID := peerId

	t.Run("Context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		info, err := testClient.NodeInfo(ctx, peerID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "context canceled")
		require.Empty(t, info)
	})

	t.Run("Context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		time.Sleep(2 * time.Millisecond)

		info, err := testClient.NodeInfo(ctx, peerID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "context deadline exceeded")
		require.Empty(t, info)
	})
}

// func TestNodeInfoConcurrent(t *testing.T) {
// 	numRequests := 10
// 	peerID := peerId

// 	var wg sync.WaitGroup
// 	wg.Add(numRequests)

// 	for i := 0; i < numRequests; i++ {
// 		go func() {
// 			defer wg.Done()
// 			ctx := context.Background()
// 			info, err := testClient.NodeInfo(ctx, peerID) // use an actual valid peer id
// 			require.NoError(t, err)
// 			require.NotEmpty(t, info.ID)
// 			require.NotEmpty(t, info.Addresses)
// 		}()
// 	}

// 	wg.Wait()
// }

func TestListConnectedNodes(t *testing.T) {
	t.Run("List connected nodes", func(t *testing.T) {
		ctx := context.Background()
		nodes, err := testClient.ListConnectedNodes(ctx)
		require.NoError(t, err)
		require.NotNil(t, nodes)
		require.GreaterOrEqual(t, len(nodes), 1)

		for _, node := range nodes {
			require.NotEmpty(t, node.ID)
			require.NotEmpty(t, node.Address)
			require.NotEmpty(t, node.Direction)
			require.GreaterOrEqual(t, node.Latency, int64(0))
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		nodes, err := testClient.ListConnectedNodes(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "context canceled")
		require.Nil(t, nodes)
	})

	t.Run("Context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		time.Sleep(2 * time.Millisecond)

		nodes, err := testClient.ListConnectedNodes(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "context deadline exceeded")
		require.Nil(t, nodes)
	})
}

func TestListConnectedNodesConcurrent(t *testing.T) {
	numRequests := 10
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			ctx := context.Background()
			nodes, err := testClient.ListConnectedNodes(ctx)
			require.NoError(t, err)
			require.NotNil(t, nodes)
		}()
	}

	wg.Wait()
}
