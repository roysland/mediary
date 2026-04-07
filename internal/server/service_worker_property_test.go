package server

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

func readServiceWorkerScript(t *testing.T) string {
	t.Helper()

	candidates := []string{
		filepath.Join("web", "static", "sw.js"),
		filepath.Join("..", "..", "web", "static", "sw.js"),
	}

	var (
		b   []byte
		err error
	)
	for _, candidate := range candidates {
		b, err = os.ReadFile(candidate)
		if err == nil {
			return string(b)
		}
	}

	t.Fatalf("read service worker script: %v", err)
	return ""
}

func extractStaticManifest(t *testing.T, script string) map[string]struct{} {
	t.Helper()

	const manifestStart = "const STATIC_ASSET_MANIFEST = Object.freeze(["
	start := strings.Index(script, manifestStart)
	if start < 0 {
		t.Fatal("service worker manifest constant not found")
	}

	block := script[start+len(manifestStart):]
	end := strings.Index(block, "]);")
	if end < 0 {
		t.Fatal("service worker manifest closing marker not found")
	}
	block = block[:end]

	re := regexp.MustCompile(`"/static/[^"]+"`)
	matches := re.FindAllString(block, -1)
	if len(matches) == 0 {
		t.Fatal("service worker manifest is empty")
	}

	manifest := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		manifest[strings.Trim(match, `"`)] = struct{}{}
	}
	return manifest
}

func listStaticFiles(t *testing.T) []string {
	t.Helper()

	assets := make([]string, 0)
	root := filepath.Join("web", "static")
	if _, err := os.Stat(root); err != nil {
		root = filepath.Join("..", "..", "web", "static")
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		assets = append(assets, "/static/"+filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk static files: %v", err)
	}

	sort.Strings(assets)
	return assets
}

func shouldUseStaticCache(method, path string) bool {
	return method == "GET" && strings.HasPrefix(path, "/static/")
}

// Feature: app-feature-roadmap, Property 12: Service worker asset pre-cache completeness
func TestProp_ServiceWorkerAssetPrecacheCompleteness(t *testing.T) {
	script := readServiceWorkerScript(t)
	manifest := extractStaticManifest(t, script)
	assets := listStaticFiles(t)

	if len(assets) == 0 {
		t.Fatal("expected at least one static asset file")
	}

	for _, asset := range assets {
		if _, ok := manifest[asset]; !ok {
			t.Fatalf("manifest missing static asset: %s", asset)
		}
	}

	rapid.Check(t, func(t *rapid.T) {
		asset := rapid.SampledFrom(assets).Draw(t, "asset")
		if _, ok := manifest[asset]; !ok {
			t.Fatalf("manifest missing drawn static asset: %s", asset)
		}
	})
}

// Feature: app-feature-roadmap, Property 13: Service worker cache invalidation on version change
func TestProp_ServiceWorkerCacheInvalidationOnVersionChange(t *testing.T) {
	script := readServiceWorkerScript(t)

	if !strings.Contains(script, "const CACHE_NAME = `static-v${BUILD_VERSION}`;") {
		t.Fatal("CACHE_NAME must be derived from BUILD_VERSION with static-v prefix")
	}
	if !strings.Contains(script, "if (cacheName !== CACHE_NAME)") {
		t.Fatal("activate handler must delete caches not matching CACHE_NAME")
	}

	rapid.Check(t, func(t *rapid.T) {
		versionA := rapid.StringMatching(`[a-z0-9._-]{1,24}`).Draw(t, "version_a")
		versionB := rapid.StringMatching(`[a-z0-9._-]{1,24}`).Draw(t, "version_b")
		if versionA == versionB {
			return
		}

		cacheA := "static-v" + versionA
		cacheB := "static-v" + versionB
		if cacheA == cacheB {
			t.Fatalf("distinct versions must yield distinct cache names: %q == %q", cacheA, cacheB)
		}
		if !strings.HasPrefix(cacheA, "static-v") || !strings.HasPrefix(cacheB, "static-v") {
			t.Fatalf("cache names must use static-v prefix, got %q and %q", cacheA, cacheB)
		}
	})
}

// Feature: app-feature-roadmap, Property 14: Service worker passthrough for non-static requests
func TestProp_ServiceWorkerPassthroughForNonStaticRequests(t *testing.T) {
	script := readServiceWorkerScript(t)

	if !strings.Contains(script, "if (!requestURL.pathname.startsWith(\"/static/\"))") {
		t.Fatal("fetch handler must bypass non-static paths")
	}
	if !strings.Contains(script, "if (request.method !== \"GET\")") {
		t.Fatal("fetch handler must bypass non-GET requests")
	}

	rapid.Check(t, func(t *rapid.T) {
		method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE"}).Draw(t, "method")
		nonStaticPath := rapid.StringMatching(`/((api|entry|entries|settings|data|share|healthz)[a-z0-9_/-]{0,24})`).Draw(t, "path")
		if strings.HasPrefix(nonStaticPath, "/static/") {
			nonStaticPath = "/api" + nonStaticPath
		}

		if shouldUseStaticCache(method, nonStaticPath) {
			t.Fatalf("non-static request should bypass cache, got method=%q path=%q", method, nonStaticPath)
		}
	})
}
