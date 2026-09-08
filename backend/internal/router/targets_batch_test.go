package router

import "testing"

func TestTargetBatchValidation(t *testing.T) {
	zero := uint(0)
	for _, in := range []targetBatchInput{{}, {URLs: []string{"https://example.com/a.png", " https://example.com/a.png "}}, {URLs: []string{"javascript:alert(1)"}}, {URLs: []string{"https://user:pass@example.com/a"}}, {URLs: []string{"https://example.com"}, ScanLimit: &zero}, {URLs: make([]string, 101)}} {
		if validateTargetBatch(in) == nil {
			t.Fatal("invalid batch accepted")
		}
	}
	if err := validateTargetBatch(targetBatchInput{URLs: []string{"https://example.com/a.png", "http://localhost:18080/b.png"}}); err != nil {
		t.Fatal(err)
	}
	if err := validateTargetBatch(targetBatchInput{URLs: []string{"/uploads/managed.png"}}); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"/uploads/../secret", "/uploads/a.png?redirect=x", "/assets/a.png"} {
		if validTargetURL(raw) {
			t.Fatalf("unsafe managed target accepted: %s", raw)
		}
	}
}
