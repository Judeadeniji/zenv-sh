package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// maxSecretMetadataBytes caps plaintext JSON stored on the server (not zero-knowledge).
const maxSecretMetadataBytes = 8192

// normalizeSecretMetadata validates and minifies metadata for storage.
// Allowed keys: mime_type, description, tags, labels. Unknown keys are rejected.
func normalizeSecretMetadata(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return []byte("{}"), nil
	}
	if len(raw) > maxSecretMetadataBytes {
		return nil, fmt.Errorf("metadata exceeds %d bytes", maxSecretMetadataBytes)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return []byte("{}"), nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		switch k {
		case "mime_type":
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return nil, fmt.Errorf("mime_type: %w", err)
			}
			if len(s) > 256 {
				return nil, errors.New("mime_type too long (max 256)")
			}
			out[k] = s
		case "description":
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return nil, fmt.Errorf("description: %w", err)
			}
			if len(s) > 4000 {
				return nil, errors.New("description too long (max 4000)")
			}
			out[k] = s
		case "tags":
			var tags []string
			if err := json.Unmarshal(v, &tags); err != nil {
				return nil, fmt.Errorf("tags: %w", err)
			}
			if len(tags) > 64 {
				return nil, errors.New("too many tags (max 64)")
			}
			for i, t := range tags {
				t = strings.TrimSpace(t)
				if len(t) > 128 {
					return nil, fmt.Errorf("tag %d too long (max 128)", i)
				}
				tags[i] = t
			}
			out[k] = tags
		case "labels":
			var labels map[string]string
			if err := json.Unmarshal(v, &labels); err != nil {
				return nil, fmt.Errorf("labels: %w", err)
			}
			if len(labels) > 64 {
				return nil, errors.New("too many labels (max 64)")
			}
			for lk, lv := range labels {
				if len(lk) > 128 {
					return nil, fmt.Errorf("label key %q too long (max 128)", lk)
				}
				if len(lv) > 512 {
					return nil, fmt.Errorf("label value for %q too long (max 512)", lk)
				}
			}
			out[k] = labels
		default:
			return nil, fmt.Errorf("unknown metadata field %q", k)
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	if len(b) > maxSecretMetadataBytes {
		return nil, fmt.Errorf("metadata exceeds %d bytes after normalization", maxSecretMetadataBytes)
	}
	return b, nil
}

// mergeSecretMetadataJSON shallow-merges patch into existing JSON.
// JSON null for a key removes that key. Unknown keys are rejected after merge.
func mergeSecretMetadataJSON(existing string, patch []byte) ([]byte, error) {
	if len(patch) == 0 {
		return normalizeSecretMetadata(nil)
	}
	if len(patch) > maxSecretMetadataBytes {
		return nil, fmt.Errorf("metadata patch exceeds %d bytes", maxSecretMetadataBytes)
	}
	base := map[string]json.RawMessage{}
	if existing != "" && existing != "{}" {
		if err := json.Unmarshal([]byte(existing), &base); err != nil {
			return nil, fmt.Errorf("existing metadata: %w", err)
		}
	}
	var p map[string]json.RawMessage
	if err := json.Unmarshal(patch, &p); err != nil {
		return nil, err
	}
	for k, v := range p {
		if string(v) == "null" {
			delete(base, k)
			continue
		}
		base[k] = v
	}
	if len(base) == 0 {
		return []byte("{}"), nil
	}
	merged, err := json.Marshal(base)
	if err != nil {
		return nil, err
	}
	return normalizeSecretMetadata(merged)
}
