package openai

import "strings"

// ModelUnavailableMessage is intentionally generic so clients do not infer
// upstream routing or internal product names from blocked aliases.
const ModelUnavailableMessage = "model is not available"

// IsUnsupportedPublicModel blocks model aliases that look available to clients
// but are not sold/provisioned by this deployment. The most important case is
// "GPT-5.5 pro": it can be manually typed by clients and may otherwise fall
// through to gpt-5.5 routing while being billed under the misleading pro alias.
func IsUnsupportedPublicModel(modelID string) bool {
	normalized := normalizePublicModelAlias(modelID)
	if normalized == "" {
		return false
	}

	return isUnsupportedProAlias(normalized)
}

// IsImageGenerationModel reports whether a model ID belongs to the OpenAI
// dedicated image-generation family exposed by /v1/models.
func IsImageGenerationModel(modelID string) bool {
	return strings.HasPrefix(normalizePublicModelAlias(modelID), "gpt-image-")
}

// FilterPublicModelIDs removes unsupported public aliases from /v1/models
// responses while preserving the original order of the remaining models.
func FilterPublicModelIDs(modelIDs []string) []string {
	return FilterPublicModelIDsForCapabilities(modelIDs, true)
}

// FilterPublicModelIDsForCapabilities also hides image models when the current
// group is not allowed to use image generation.
func FilterPublicModelIDsForCapabilities(modelIDs []string, allowImageGeneration bool) []string {
	if len(modelIDs) == 0 {
		return modelIDs
	}
	filtered := make([]string, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		if IsUnsupportedPublicModel(modelID) {
			continue
		}
		if !allowImageGeneration && IsImageGenerationModel(modelID) {
			continue
		}
		filtered = append(filtered, modelID)
	}
	return filtered
}

// FilterPublicModels removes unsupported public aliases from the default
// OpenAI model list while preserving model metadata for allowed entries.
func FilterPublicModels(models []Model) []Model {
	return FilterPublicModelsForCapabilities(models, true)
}

// FilterPublicModelsForCapabilities removes unsupported aliases from the
// default OpenAI model list and optionally hides image models.
func FilterPublicModelsForCapabilities(models []Model, allowImageGeneration bool) []Model {
	if len(models) == 0 {
		return models
	}
	filtered := make([]Model, 0, len(models))
	for _, model := range models {
		if IsUnsupportedPublicModel(model.ID) {
			continue
		}
		if !allowImageGeneration && IsImageGenerationModel(model.ID) {
			continue
		}
		filtered = append(filtered, model)
	}
	return filtered
}

func isUnsupportedProAlias(normalizedModelID string) bool {
	if normalizedModelID == "gpt-5.2-pro" || strings.HasPrefix(normalizedModelID, "gpt-5.2-pro-") {
		return true
	}
	if normalizedModelID == "gpt-5.4-pro" || strings.HasPrefix(normalizedModelID, "gpt-5.4-pro-") {
		return true
	}
	return normalizedModelID == "gpt-5.5-pro" || strings.HasPrefix(normalizedModelID, "gpt-5.5-pro-")
}

func normalizePublicModelAlias(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ""
	}
	if idx := strings.LastIndex(modelID, "/"); idx >= 0 && idx+1 < len(modelID) {
		modelID = modelID[idx+1:]
	}
	modelID = strings.ToLower(strings.TrimSpace(modelID))
	replacer := strings.NewReplacer(
		"_", "-",
		" ", "-",
		"\t", "-",
		"\n", "-",
		"\r", "-",
	)
	modelID = replacer.Replace(modelID)
	for strings.Contains(modelID, "--") {
		modelID = strings.ReplaceAll(modelID, "--", "-")
	}
	return strings.Trim(modelID, "-")
}
