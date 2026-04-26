package db

import "time"

func (d *DB) CreateJob(j Job) error {
	_, err := d.Exec(
		"INSERT INTO jobs(id,name,client_name,status,mode,created_at) VALUES(?,?,?,?,?,?)",
		j.ID, j.Name, j.ClientName, j.Status, j.Mode, j.CreatedAt,
	)
	return err
}

func (d *DB) UpdateJobStatus(id, status string) error {
	_, err := d.Exec("UPDATE jobs SET status=? WHERE id=?", status, id)
	return err
}

func (d *DB) FinishJob(id, status string) error {
	_, err := d.Exec("UPDATE jobs SET status=?,finished_at=? WHERE id=?", status, time.Now(), id)
	return err
}

func (d *DB) FailJob(id, message string) error {
	_, err := d.Exec("UPDATE jobs SET status='failed',finished_at=?,error_message=? WHERE id=?", time.Now(), message, id)
	return err
}

func (d *DB) GetJob(id string) (Job, error) {
	var j Job
	var finishedAt *time.Time
	var errMsg *string
	err := d.QueryRow(
		"SELECT id,name,client_name,status,mode,created_at,finished_at,COALESCE(error_message,'') FROM jobs WHERE id=?", id,
	).Scan(&j.ID, &j.Name, &j.ClientName, &j.Status, &j.Mode, &j.CreatedAt, &finishedAt, &j.ErrorMessage)
	j.FinishedAt = finishedAt
	_ = errMsg
	return j, err
}

func (d *DB) ListJobs() ([]Job, error) {
	rows, err := d.Query("SELECT id,name,client_name,status,mode,created_at,finished_at,COALESCE(error_message,'') FROM jobs ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		var j Job
		var finishedAt *time.Time
		if err := rows.Scan(&j.ID, &j.Name, &j.ClientName, &j.Status, &j.Mode, &j.CreatedAt, &finishedAt, &j.ErrorMessage); err != nil {
			return nil, err
		}
		j.FinishedAt = finishedAt
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// ClearHistory deletes all terminal jobs and their child rows.
// The schema has no ON DELETE CASCADE, so child rows are deleted explicitly first.
// Active jobs (running/provisioning) are left untouched.
func (d *DB) ClearHistory() error {
	terminal := `SELECT id FROM jobs WHERE status IN ('done','failed','cancelled')`
	for _, table := range []string{"results", "proxmox_snapshots", "proxmox_vm_snapshots", "workers"} {
		_, err := d.Exec(`DELETE FROM ` + table + ` WHERE job_id IN (` + terminal + `)`)
		if err != nil {
			return err
		}
	}
	_, err := d.Exec(`DELETE FROM jobs WHERE status IN ('done','failed','cancelled')`)
	return err
}

// ListActiveJobs retourne les jobs en cours (running ou provisioning).
func (d *DB) ListActiveJobs() ([]Job, error) {
	rows, err := d.Query(
		`SELECT id,name,client_name,status,mode,created_at,finished_at,COALESCE(error_message,'')
		 FROM jobs WHERE status IN ('running','provisioning') ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		var j Job
		var finishedAt *time.Time
		if err := rows.Scan(&j.ID, &j.Name, &j.ClientName, &j.Status, &j.Mode, &j.CreatedAt, &finishedAt, &j.ErrorMessage); err != nil {
			return nil, err
		}
		j.FinishedAt = finishedAt
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// ListRecentJobs retourne les N derniers jobs termines (done/failed/cancelled).
func (d *DB) ListRecentJobs(n int) ([]Job, error) {
	rows, err := d.Query(
		`SELECT id,name,client_name,status,mode,created_at,finished_at,COALESCE(error_message,'')
		 FROM jobs WHERE status IN ('done','failed','cancelled')
		 ORDER BY finished_at DESC LIMIT ?`,
		n,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []Job
	for rows.Next() {
		var j Job
		var finishedAt *time.Time
		if err := rows.Scan(&j.ID, &j.Name, &j.ClientName, &j.Status, &j.Mode, &j.CreatedAt, &finishedAt, &j.ErrorMessage); err != nil {
			return nil, err
		}
		j.FinishedAt = finishedAt
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}
