package openai

import "testing"

func TestIsUnsupportedPublicModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{model: "gpt-5.5", want: false},
		{model: "gpt-5.5-pro", want: true},
		{model: "GPT-5.5 Pro", want: true},
		{model: "codex/GPT-5.5 pro", want: true},
		{model: "gpt-5.5-pro-2026-05-01", want: true},
		{model: "gpt-5.4-pro", want: true},
		{model: "gpt-5.2-pro", want: true},
		{model: "gpt-5.2-pro-2025-12-11", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := IsUnsupportedPublicModel(tt.model); got != tt.want {
				t.Fatalf("IsUnsupportedPublicModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestFilterPublicModelIDs(t *testing.T) {
	input := []string{"gpt-5.5", "gpt-5.5-pro", "codex/GPT-5.5 pro", "gpt-5.4"}
	got := FilterPublicModelIDs(input)
	want := []string{"gpt-5.5", "gpt-5.4"}
	if len(got) != len(want) {
		t.Fatalf("len(FilterPublicModelIDs) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("FilterPublicModelIDs[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFilterPublicModels(t *testing.T) {
	input := []Model{
		{ID: "gpt-5.5", DisplayName: "GPT-5.5"},
		{ID: "gpt-5.5-pro", DisplayName: "GPT-5.5 Pro"},
		{ID: "gpt-5.4", DisplayName: "GPT-5.4"},
	}
	got := FilterPublicModels(input)
	if len(got) != 2 {
		t.Fatalf("len(FilterPublicModels) = %d, want 2: %#v", len(got), got)
	}
	if got[0].ID != "gpt-5.5" || got[1].ID != "gpt-5.4" {
		t.Fatalf("FilterPublicModels returned unexpected IDs: %#v", got)
	}
	if got[0].DisplayName != "GPT-5.5" {
		t.Fatalf("FilterPublicModels should preserve metadata, got display name %q", got[0].DisplayName)
	}
}

func TestFilterPublicModelIDsForCapabilities_HidesImagesWhenDisabled(t *testing.T) {
	input := []string{"gpt-5.5", "gpt-image-2", "gpt-image-1.5"}
	got := FilterPublicModelIDsForCapabilities(input, false)
	want := []string{"gpt-5.5"}
	if len(got) != len(want) {
		t.Fatalf("len(FilterPublicModelIDsForCapabilities) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("FilterPublicModelIDsForCapabilities[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFilterPublicModelsForCapabilities_AllowsImagesWhenEnabled(t *testing.T) {
	input := []Model{
		{ID: "gpt-5.5"},
		{ID: "gpt-image-2"},
	}
	got := FilterPublicModelsForCapabilities(input, true)
	if len(got) != 2 {
		t.Fatalf("len(FilterPublicModelsForCapabilities) = %d, want 2: %#v", len(got), got)
	}
	if got[1].ID != "gpt-image-2" {
		t.Fatalf("image model should be preserved when enabled, got %#v", got)
	}
}
