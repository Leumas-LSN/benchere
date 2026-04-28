const BASE = '/api'

async function request(method, path, body) {
  const hasBody = body !== undefined
  const opts = {
    method,
    headers: hasBody ? { 'Content-Type': 'application/json' } : {},
    body: hasBody ? JSON.stringify(body) : undefined,
  }
  const res = await fetch(BASE + path, opts)
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`${res.status}: ${text}`)
  }
  if (res.status === 204) return null
  return res.json()
}

export const api = {
  // Settings
  getSettings:  ()          => request('GET',    '/settings'),
  saveSettings: (settings)  => request('POST',   '/settings', settings),
  testProxmox:  ()          => request('POST',   '/proxmox/test'),
  scanStorages: ()          => request('GET',    '/proxmox/storages'),
  scanBridges:  ()          => request('GET',    '/proxmox/bridges'),

  // Jobs
  listJobs:   ()    => request('GET',  '/jobs'),
  getJob:     (id)  => request('GET',  `/jobs/${id}`),
  createJob:  (job) => request('POST', '/jobs', job),
  cancelJob:  (id)  => request('POST', `/jobs/${id}/cancel`),
  listWorkers: (id)  => request('GET',  `/jobs/${id}/workers`),
  clearHistory: ()  => request('DELETE', '/jobs'),

  // Profiles
  listProfiles:   ()        => request('GET',    '/profiles'),
  createProfile:  (profile) => request('POST',   '/profiles', profile),
  deleteProfile:  (id)      => request('DELETE', `/profiles/${id}`),
  updateProfile:  (id, body) => request('PUT',    `/profiles/${id}`, body),

  // Report download URLs (no fetch, used as href)
  reportPdfUrl:  (id, lang) => `/api/jobs/${id}/report.pdf?lang=${lang || 'fr'}`,
  reportHtmlUrl: (id, lang) => `/api/jobs/${id}/report.html?lang=${lang || 'fr'}`,

  // Overview + results
  getOverview:    ()   => request('GET', '/overview'),
  getVersion:     ()   => request('GET', '/version'),
  getJobResults:  (id) => request('GET', `/jobs/${id}/results`),

  // CSV export URL helper.
  exportCsvUrl:   (id) => `/api/jobs/${id}/results.csv`,

  // Debug bundle URL helper. Returns a tar.gz with the DB snapshot,
  // scrubbed settings, journalctl, raw elbencho and ansible logs, worker
  // sysinfo and a best-effort proxmox plus ceph snapshot. The endpoint
  // replies 409 when the job is still in flight, so the UI button is
  // disabled in that case.
  debugBundleUrl: (id) => `/api/jobs/${id}/debug`,
}
