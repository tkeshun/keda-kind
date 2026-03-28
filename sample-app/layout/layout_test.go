package layout_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSampleAppLayoutUsesSampleAppPaths(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))

	requiredFiles := []string{
		"sample-app/docker/enqueue.Dockerfile",
		"sample-app/docker/dequeue.Dockerfile",
		"manifest/enqueue-app/Chart.yaml",
		"manifest/dequeue-app/Chart.yaml",
	}

	for _, relativePath := range requiredFiles {
		if _, err := os.Stat(filepath.Join(root, relativePath)); err != nil {
			t.Fatalf("expected %s to exist: %v", relativePath, err)
		}
	}
}

func TestMakefileReferencesSampleAppAssets(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))

	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}

	content := string(makefile)
	requiredSnippets := []string{
		"sample-app/docker/enqueue.Dockerfile",
		"sample-app/docker/dequeue.Dockerfile",
		"./manifest/enqueue-app",
		"./manifest/dequeue-app",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected Makefile to reference %q", snippet)
		}
	}
}

func TestComposeReferencesSampleAppDockerfiles(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))

	composeFile, err := os.ReadFile(filepath.Join(root, "compose.yaml"))
	if err != nil {
		t.Fatalf("read compose.yaml: %v", err)
	}

	content := string(composeFile)
	requiredSnippets := []string{
		"dockerfile: sample-app/docker/enqueue.Dockerfile",
		"dockerfile: sample-app/docker/dequeue.Dockerfile",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected compose.yaml to reference %q", snippet)
		}
	}
}
