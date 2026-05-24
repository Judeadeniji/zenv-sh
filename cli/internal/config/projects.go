package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var projectIDPattern = regexp.MustCompile(`^[0-9a-fA-F-]{36}$`)

// ProjectsDir is ~/.config/zenv/projects (one credentials file per project).
func ProjectsDir() string {
	return filepath.Join(Dir(), "projects")
}

func projectCredsPath(projectID string) (string, error) {
	if !projectIDPattern.MatchString(projectID) {
		return "", fmt.Errorf("invalid project ID %q", projectID)
	}
	return filepath.Join(ProjectsDir(), projectID, "credentials"), nil
}

func loadProjectCredentials(projectID string) map[string]string {
	path, err := projectCredsPath(projectID)
	if err != nil {
		return map[string]string{}
	}
	return loadKV(path)
}

// SetForProject stores a secret for a specific project (preferred over global).
func SetForProject(projectID, key, value string) error {
	if !isCredential(key) {
		return fmt.Errorf("%s cannot be stored per-project (use .zenv or global config)", key)
	}
	path, err := projectCredsPath(projectID)
	if err != nil {
		return err
	}
	return setKV(path, key, value)
}

// GetForProject reads a secret for a specific project.
func GetForProject(projectID, key string) string {
	return loadProjectCredentials(projectID)[key]
}

// UnsetForProject removes a secret from a project's credentials file.
func UnsetForProject(projectID, key string) error {
	path, err := projectCredsPath(projectID)
	if err != nil {
		return err
	}
	return removeKV(path, key)
}

// ListForProject returns all credentials for a project (token, project_key).
func ListForProject(projectID string) map[string]string {
	return loadProjectCredentials(projectID)
}

// ListConfiguredProjects returns project IDs that have a credentials directory.
func ListConfiguredProjects() ([]string, error) {
	entries, err := os.ReadDir(ProjectsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if projectIDPattern.MatchString(id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// ProjectCredentialsPath returns the credentials file path for display.
func ProjectCredentialsPath(projectID string) (string, error) {
	return projectCredsPath(projectID)
}

// HasProjectCredentials reports whether a project has any stored secrets.
func HasProjectCredentials(projectID string) bool {
	kv := loadProjectCredentials(projectID)
	return kv[KeyToken] != "" || kv[KeyProjectKey] != ""
}

// SetGlobalSecret stores a credential in the legacy global file (discouraged).
func SetGlobalSecret(key, value string) error {
	if !isCredential(key) {
		return fmt.Errorf("%s is not a credential", key)
	}
	return setKV(globalCredsPath(), key, value)
}

// migrateGlobalSecretsToProject copies global token/project_key into a project store if missing there.
func migrateGlobalSecretsToProject(projectID string) {
	global := loadGlobalCredentials()
	proj := loadProjectCredentials(projectID)
	changed := false
	for _, k := range []string{KeyToken, KeyProjectKey} {
		if global[k] != "" && proj[k] == "" {
			_ = SetForProject(projectID, k, global[k])
			changed = true
		}
	}
	if changed {
		// Leave global in place for backward compat; users can delete manually.
	}
}
