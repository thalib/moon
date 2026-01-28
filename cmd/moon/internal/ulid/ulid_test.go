package ulid

import (
	"strings"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	t.Run("Generate valid ULID", func(t *testing.T) {
		id := Generate()

		if len(id) != 26 {
			t.Errorf("expected ULID length 26, got %d", len(id))
		}

		if !IsValid(id) {
			t.Errorf("generated ULID is not valid: %s", id)
		}
	})

	t.Run("Generate unique ULIDs", func(t *testing.T) {
		id1 := Generate()
		id2 := Generate()

		if id1 == id2 {
			t.Errorf("expected unique ULIDs, got duplicates: %s", id1)
		}
	})

	t.Run("Generate sortable ULIDs", func(t *testing.T) {
		id1 := Generate()
		time.Sleep(2 * time.Millisecond)
		id2 := Generate()

		if id1 >= id2 {
			t.Errorf("expected id1 < id2 for sortability, got id1=%s, id2=%s", id1, id2)
		}
	})
}

func TestGenerateWithTime(t *testing.T) {
	t.Run("Generate with specific time", func(t *testing.T) {
		testTime := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
		id := GenerateWithTime(testTime)

		if len(id) != 26 {
			t.Errorf("expected ULID length 26, got %d", len(id))
		}

		if !IsValid(id) {
			t.Errorf("generated ULID is not valid: %s", id)
		}

		extractedTime, err := Time(id)
		if err != nil {
			t.Fatalf("failed to extract time from ULID: %v", err)
		}

		// ULID timestamps are in milliseconds, so we compare at millisecond precision
		// Allow for small timestamp differences due to ULID timestamp resolution
		timeDiff := extractedTime.Sub(testTime)
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}
		if timeDiff > time.Second {
			t.Errorf("time difference too large: expected %v, got %v (diff: %v)", testTime, extractedTime, timeDiff)
		}
	})

	t.Run("Generate with past time", func(t *testing.T) {
		pastTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		id := GenerateWithTime(pastTime)

		if !IsValid(id) {
			t.Errorf("generated ULID with past time is not valid: %s", id)
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Valid ULID",
			input:   "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			wantErr: false,
		},
		{
			name:    "Empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Too short",
			input:   "01ARZ3NDEKTSV4RRFFQ69G5FA",
			wantErr: true,
		},
		{
			name:    "Too long",
			input:   "01ARZ3NDEKTSV4RRFFQ69G5FAVX",
			wantErr: true,
		},
		{
			name:    "Random string same length",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			isValid := IsValid(tt.input)
			if isValid == tt.wantErr {
				t.Errorf("IsValid() = %v, want %v", isValid, !tt.wantErr)
			}
		})
	}
}

func TestParse(t *testing.T) {
	t.Run("Parse valid ULID", func(t *testing.T) {
		validULID := Generate()
		parsed, err := Parse(validULID)

		if err != nil {
			t.Errorf("unexpected error parsing valid ULID: %v", err)
		}

		if parsed.String() != validULID {
			t.Errorf("expected parsed ULID %s, got %s", validULID, parsed.String())
		}
	})

	t.Run("Parse invalid ULID", func(t *testing.T) {
		_, err := Parse("invalid-ulid")

		if err == nil {
			t.Error("expected error parsing invalid ULID, got nil")
		}

		if !strings.Contains(err.Error(), "invalid ULID format") {
			t.Errorf("expected error message to contain 'invalid ULID format', got: %v", err)
		}
	})
}

func TestTime(t *testing.T) {
	t.Run("Extract time from ULID", func(t *testing.T) {
		now := time.Now()
		id := GenerateWithTime(now)

		extractedTime, err := Time(id)
		if err != nil {
			t.Fatalf("failed to extract time: %v", err)
		}

		// Compare at millisecond precision
		if extractedTime.Truncate(time.Millisecond) != now.Truncate(time.Millisecond) {
			t.Errorf("expected time %v, got %v", now, extractedTime)
		}
	})

	t.Run("Extract time from invalid ULID", func(t *testing.T) {
		_, err := Time("invalid-ulid")

		if err == nil {
			t.Error("expected error extracting time from invalid ULID, got nil")
		}
	})
}

func TestCompare(t *testing.T) {
	t.Run("Compare earlier < later", func(t *testing.T) {
		earlier := GenerateWithTime(time.Now().Add(-1 * time.Hour))
		time.Sleep(1 * time.Millisecond)
		later := GenerateWithTime(time.Now())

		result, err := Compare(earlier, later)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != -1 {
			t.Errorf("expected -1 (earlier < later), got %d", result)
		}
	})

	t.Run("Compare later > earlier", func(t *testing.T) {
		earlier := GenerateWithTime(time.Now().Add(-1 * time.Hour))
		later := GenerateWithTime(time.Now())

		result, err := Compare(later, earlier)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != 1 {
			t.Errorf("expected 1 (later > earlier), got %d", result)
		}
	})

	t.Run("Compare equal ULIDs", func(t *testing.T) {
		id := Generate()

		result, err := Compare(id, id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != 0 {
			t.Errorf("expected 0 (equal), got %d", result)
		}
	})

	t.Run("Compare with invalid first ULID", func(t *testing.T) {
		valid := Generate()
		_, err := Compare("invalid", valid)

		if err == nil {
			t.Error("expected error comparing with invalid ULID, got nil")
		}
	})

	t.Run("Compare with invalid second ULID", func(t *testing.T) {
		valid := Generate()
		_, err := Compare(valid, "invalid")

		if err == nil {
			t.Error("expected error comparing with invalid ULID, got nil")
		}
	})
}

func TestIsValid(t *testing.T) {
	t.Run("Valid generated ULID", func(t *testing.T) {
		id := Generate()
		if !IsValid(id) {
			t.Errorf("generated ULID should be valid: %s", id)
		}
	})

	t.Run("Invalid empty string", func(t *testing.T) {
		if IsValid("") {
			t.Error("empty string should not be valid")
		}
	})

	t.Run("Invalid short string", func(t *testing.T) {
		if IsValid("12345") {
			t.Error("short string should not be valid")
		}
	})
}

func BenchmarkGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Generate()
	}
}

func BenchmarkValidate(b *testing.B) {
	id := Generate()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Validate(id)
	}
}

func BenchmarkParse(b *testing.B) {
	id := Generate()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Parse(id)
	}
}
