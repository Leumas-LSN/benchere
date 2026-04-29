package db

import (
	"github.com/google/uuid"
)

// Profile represents a benchmark profile stored in the DB.
type Profile struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Engine         string `json:"engine"`
	ConfigJSON     string `json:"config_json"`
	Description    string `json:"description"`
	ThresholdsJSON string `json:"thresholds_json"`
	IsBuiltin      bool   `json:"is_builtin"`
}

// ProfileThresholds are the pass/fail thresholds for a profile.
type ProfileThresholds struct {
	MinIOPSRead  float64 `json:"min_iops_read"`
	MinIOPSWrite float64 `json:"min_iops_write"`
	MaxLatencyMs float64 `json:"max_latency_ms"`
}

// ListProfiles returns all profiles ordered by engine, builtin, then name.
// Both fio and elbencho profiles are returned; the frontend filters by
// the currently selected engine when building the new-job form.
func (d *DB) ListProfiles() ([]Profile, error) {
	rows, err := d.Query(
		"SELECT id, name, engine, config_json, description, thresholds_json, is_builtin FROM benchmark_profiles ORDER BY engine ASC, is_builtin DESC, name ASC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var profiles []Profile
	for rows.Next() {
		var p Profile
		var isBuiltin int
		if err := rows.Scan(&p.ID, &p.Name, &p.Engine, &p.ConfigJSON, &p.Description, &p.ThresholdsJSON, &isBuiltin); err != nil {
			return nil, err
		}
		p.IsBuiltin = isBuiltin == 1
		profiles = append(profiles, p)
	}
	if profiles == nil {
		profiles = []Profile{}
	}
	return profiles, rows.Err()
}

// GetProfile returns a profile by ID.
func (d *DB) GetProfile(id string) (Profile, error) {
	var p Profile
	var isBuiltin int
	err := d.QueryRow(
		"SELECT id, name, engine, config_json, description, thresholds_json, is_builtin FROM benchmark_profiles WHERE id = ?", id,
	).Scan(&p.ID, &p.Name, &p.Engine, &p.ConfigJSON, &p.Description, &p.ThresholdsJSON, &isBuiltin)
	p.IsBuiltin = isBuiltin == 1
	return p, err
}

// GetProfileByName returns the elbencho profile with the given name. Kept
// for backward compatibility with callers that predate the engine column.
func (d *DB) GetProfileByName(name string) (Profile, error) {
	return d.GetProfileByNameAndEngine(name, "elbencho")
}

// GetProfileByNameAndEngine returns the profile matching both name and
// engine. The (name, engine) tuple is unique in the schema (since v1.11.0).
func (d *DB) GetProfileByNameAndEngine(name, engine string) (Profile, error) {
	var p Profile
	var isBuiltin int
	err := d.QueryRow(
		"SELECT id, name, engine, config_json, description, thresholds_json, is_builtin FROM benchmark_profiles WHERE name = ? AND engine = ?", name, engine,
	).Scan(&p.ID, &p.Name, &p.Engine, &p.ConfigJSON, &p.Description, &p.ThresholdsJSON, &isBuiltin)
	p.IsBuiltin = isBuiltin == 1
	return p, err
}

// UpdateProfile updates description and thresholds_json for a profile.
func (d *DB) UpdateProfile(id string, description, thresholdsJSON string) error {
	_, err := d.Exec(
		"UPDATE benchmark_profiles SET description = ?, thresholds_json = ? WHERE id = ?",
		description, thresholdsJSON, id,
	)
	return err
}

// CreateProfile creates a new non-builtin profile. Engine defaults to
// elbencho when empty for backward compatibility with the existing UI.
func (d *DB) CreateProfile(name, configJSON, description string) (string, error) {
	return d.CreateProfileWithEngine(name, "elbencho", configJSON, description)
}

// CreateProfileWithEngine creates a new non-builtin profile pinned to the
// given engine. Used when the API explicitly carries an engine field.
func (d *DB) CreateProfileWithEngine(name, engine, configJSON, description string) (string, error) {
	if engine == "" {
		engine = "elbencho"
	}
	id := uuid.NewString()
	_, err := d.Exec(
		"INSERT INTO benchmark_profiles (id, name, engine, config_json, description, thresholds_json, is_builtin) VALUES (?, ?, ?, ?, ?, \"\", 0)",
		id, name, engine, configJSON, description,
	)
	if err != nil {
		return "", err
	}
	return id, nil
}

// DeleteProfile deletes a non-builtin profile (builtin profiles are protected).
func (d *DB) DeleteProfile(id string) error {
	_, err := d.Exec("DELETE FROM benchmark_profiles WHERE id = ? AND is_builtin = 0", id)
	return err
}
