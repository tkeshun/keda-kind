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
		"manifest/argocd/Chart.yaml",
		"manifest/infra-bundle/Chart.yaml",
		"argocd/applicationsets/env-bundle.yaml",
		"argocd/namespaces/sample-applicationset.yaml",
		"argocd/projects/sample-app.yaml",
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
		"argocd/applicationsets/env-bundle.yaml",
		"argocd/namespaces/sample-applicationset.yaml",
		"argocd/projects/sample-app.yaml",
		"helm-deps-argocd:",
		"install-argocd:",
		"argocd-ready:",
		"helm upgrade --install enqueue-app",
		"helm upgrade --install dequeue-app",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected Makefile to reference %q", snippet)
		}
	}

	makeTargets := []string{
		"install-enqueue:",
		"install-enqueue-http:",
	}
	for _, target := range makeTargets {
		if !strings.Contains(content, target) {
			t.Fatalf("expected Makefile to define target %q", target)
		}
	}
	for _, snippet := range []string{
		"helm upgrade enqueue-app",
		"helm uninstall --ignore-not-found enqueue-app",
		"helm uninstall --ignore-not-found dequeue-app",
	} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected Makefile to reference %q", snippet)
		}
	}
}

func TestReadmeMentionsArgoCDFlow(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}

	content := string(readme)
	requiredSnippets := []string{
		"manifest/argocd",
		"argocd/applicationsets/env-bundle.yaml",
		"argocd/namespaces/sample-applicationset.yaml",
		"argocd/projects/sample-app.yaml",
		"make install-argocd",
		"make argocd-ready",
		"enqueue-app",
		"dequeue-app",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected README.md to reference %q", snippet)
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

func TestApplicationSetEnablesServerSideApplyForKEDAOperator(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))

	applicationSet, err := os.ReadFile(filepath.Join(root, "argocd", "applicationsets", "env-bundle.yaml"))
	if err != nil {
		t.Fatalf("read ApplicationSet: %v", err)
	}

	content := string(applicationSet)
	if !strings.Contains(content, "name: keda-operator") {
		t.Fatalf("expected ApplicationSet to define keda-operator")
	}
	for _, snippet := range []string{
		"name: enqueue-app",
		"path: \"{{ .chartPath }}\"",
		"releaseName: \"{{ .releaseName }}\"",
		"name: dequeue-app",
		"chartPath: manifest/enqueue-app",
		"chartPath: manifest/dequeue-app",
	} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("expected ApplicationSet to reference %q", snippet)
		}
	}
	if strings.Contains(content, "manifest/app-bundle") {
		t.Fatalf("expected ApplicationSet to stop referencing manifest/app-bundle")
	}
	if !strings.Contains(content, "serverSideApply: 'true'") {
		t.Fatalf("expected keda-operator generator element to enable server-side apply")
	}
	if !strings.Contains(content, "- ServerSideApply={{ .serverSideApply }}") {
		t.Fatalf("expected ApplicationSet sync options to template ServerSideApply")
	}
}
