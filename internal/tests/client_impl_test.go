package ipfs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	mocked "github.com/zde37/Hive/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestClientImpl_Add(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "ipfs_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFilePath := filepath.Join(tempDir, "test_file.txt")
	err = os.WriteFile(testFilePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create a test directory
	testDirPath := filepath.Join(tempDir, "test_dir")
	err = os.Mkdir(testDirPath, 0755)
	require.NoError(t, err)

	// Create a file inside the test directory
	testDirFilePath := filepath.Join(testDirPath, "test_file_in_dir.txt")
	err = os.WriteFile(testDirFilePath, []byte("test content in directory"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name       string
		fileName   string
		filePath   string
		buildStubs func(client *mocked.MockClient)
	}{
		{
			name:     "Add file successfully",
			fileName: "test_file.txt",
			filePath: testFilePath,
			buildStubs: func(client *mocked.MockClient) {
				client.EXPECT().
					Add(gomock.Any(), gomock.Eq("test_file.txt"), gomock.Eq(testFilePath)).
					Times(1).
					Return("test_file.txt", "test_file.txt", nil)
			},
		},
		{
			name:     "Add directory successfully",
			fileName: "test_dir",
			filePath: testDirPath,
			buildStubs: func(client *mocked.MockClient) {
				client.EXPECT().
					Add(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).
					Return("test_dir", "test_dir", nil)
			},
		},
		{
			name:     "Empty file name",
			fileName: "",
			filePath: testFilePath,
			buildStubs: func(client *mocked.MockClient) {
				client.EXPECT().
					Add(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
		},
		{
			name:     "Empty file path",
			fileName: "test_file.txt",
			filePath: "",
			buildStubs: func(client *mocked.MockClient) {
				client.EXPECT().
					Add(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
		},
		{
			name:     "Non-existent file",
			fileName: "non_existent.txt",
			filePath: filepath.Join(tempDir, "non_existent.txt"),
			buildStubs: func(client *mocked.MockClient) {
				client.EXPECT().
					Add(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			service := mocked.NewMockClient(ctrl)
			tt.buildStubs(service)
		})
	}
}
