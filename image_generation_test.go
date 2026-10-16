package main

import (
	"io"
	"testing"
)

func TestImageProviderReadsBaiConfigAndReusesReportKey(t *testing.T) {
	config := []byte("report_model:\n  provider: bai\n  api_key: test-key\n  model: gpt-5.2\nimage_model:\n  provider: bai\n  model: gpt-image-2\n  size: 2528x1696\n  quality: high\n")
	values := imageModelYAMLValues(string(config))
	if values["model"] != "gpt-image-2" || values["size"] != "2528x1696" {
		t.Fatalf("image YAML values = %#v", values)
	}
	provider, err := newImageProvider(values["provider"], "test-key", values["model"], values["size"], values["quality"])
	if err != nil {
		t.Fatalf("newImageProvider() error = %v", err)
	}
	if provider.Endpoint != "https://api.b.ai/v1/images/generations" || provider.Model != "gpt-image-2" || provider.Size != "2528x1696" {
		t.Fatalf("unexpected image provider: %#v", provider)
	}
}

func TestImageGenerationHelpersKeepInputsSafe(t *testing.T) {
	if got := compactBase64(" YWJj\nZA== "); got != "YWJjZA==" {
		t.Fatalf("compactBase64() = %q", got)
	}
	if got := safeImageName("RPRD: 中国 Agent / v1"); got == "" || got == "RPRD: 中国 Agent / v1" {
		t.Fatalf("safeImageName() = %q", got)
	}
	if !retryableModelTransportError(io.EOF) {
		t.Fatal("EOF must be retryable")
	}
}
