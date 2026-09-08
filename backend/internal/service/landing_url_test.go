package service

import "testing"

func TestManagedUploadURLNormalization(t *testing.T) {
	for input, want := range map[string]string{
		"http://127.0.0.1:18080/uploads/group.png?old=1": "/uploads/group.png",
		"http://localhost:18080/uploads/group.png":       "/uploads/group.png",
		"https://cdn.example.com/uploads/group.png":      "https://cdn.example.com/uploads/group.png",
		"/uploads/group.png":                             "/uploads/group.png",
	} {
		if got := normalizeManagedUploadURL(input); got != want {
			t.Fatalf("normalizeManagedUploadURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestImageURLUsesPathInsteadOfQuery(t *testing.T) {
	if !isImageURL("/uploads/group.png?v=1") {
		t.Fatal("managed PNG with query was not recognized")
	}
	if isImageURL("https://example.com/download?name=group.png") {
		t.Fatal("non-image path was recognized from query text")
	}
}
