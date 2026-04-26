package db

func (d *DB) CreateWorker(w Worker) error {
	_, err := d.Exec(
		"INSERT INTO workers(id,job_id,vm_id,proxmox_node,ip,status) VALUES(?,?,?,?,?,?)",
		w.ID, w.JobID, w.VMID, w.ProxmoxNode, w.IP, w.Status,
	)
	return err
}

func (d *DB) UpdateWorkerIP(id, ip string) error {
	_, err := d.Exec("UPDATE workers SET ip=? WHERE id=?", ip, id)
	return err
}

func (d *DB) UpdateWorkerStatus(id, status string) error {
	_, err := d.Exec("UPDATE workers SET status=? WHERE id=?", status, id)
	return err
}

func (d *DB) ListWorkersByJob(jobID string) ([]Worker, error) {
	rows, err := d.Query("SELECT id,job_id,vm_id,proxmox_node,ip,status FROM workers WHERE job_id=?", jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var workers []Worker
	for rows.Next() {
		var w Worker
		if err := rows.Scan(&w.ID, &w.JobID, &w.VMID, &w.ProxmoxNode, &w.IP, &w.Status); err != nil {
			return nil, err
		}
		workers = append(workers, w)
	}
	return workers, rows.Err()
}
