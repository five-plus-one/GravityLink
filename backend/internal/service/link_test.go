package service

import (
	"testing"

	"gravitylink/backend/internal/cache"
)

func TestAppendUTMDoesNotOverrideExistingValues(t *testing.T) {
	link := cache.CachedLink{
		UTMSource:   "wechat",
		UTMMedium:   "social",
		UTMCampaign: "spring",
	}

	actual := appendUTM("https://example.com/page?utm_source=existing&foo=bar", link)
	expected := "https://example.com/page?foo=bar&utm_campaign=spring&utm_medium=social&utm_source=existing"
	if actual != expected {
		t.Fatalf("appendUTM() = %q, want %q", actual, expected)
	}
}

func TestAppendUTMSkipsEmptyValues(t *testing.T) {
	link := cache.CachedLink{UTMContent: "poster_a"}

	actual := appendUTM("https://example.com/page", link)
	expected := "https://example.com/page?utm_content=poster_a"
	if actual != expected {
		t.Fatalf("appendUTM() = %q, want %q", actual, expected)
	}
}
