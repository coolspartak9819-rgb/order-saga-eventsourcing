const api = async (path, options) => { const response = await fetch(path, options); const body = await response.json(); if (!response.ok) throw new Error(body.error || 'Request failed'); return body; };
const $ = selector => document.querySelector(selector);
let statusFilter = '';

function escapeHtml(value) { return String(value ?? '').replace(/[&<>"']/g, ch => ({ '&':'&amp;', '<':'&lt;', '>':'&gt;', '"':'&quot;', "'":'&#039;' }[ch])); }
function showToast(message) { const toast = $('#toast'); toast.textContent = message; toast.classList.add('visible'); setTimeout(() => toast.classList.remove('visible'), 2200); }
function time(value) { return new Date(value).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }); }
function payload(value) {
  try { return JSON.stringify(typeof value === 'string' ? JSON.parse(value) : value, null, 2); } catch { return String(value ?? ''); }
}

async function refresh() {
  try {
    const [stats, result, queue] = await Promise.all([api('/api/stats'), api(`/api/jobs${statusFilter ? `?status=${statusFilter}` : ''}`), api('/api/queue')]);
    const statuses = ['PENDING', 'RUNNING', 'COMPLETED', 'FAILED'];
    $('#stats').innerHTML = statuses.map(status => `<div class="stat"><label>${status.toLowerCase()}</label><strong>${stats[status] || 0}</strong></div>`).join('') + `<div class="stat"><label>queue pending</label><strong>${queue.pending}</strong></div>`;
    $('#jobs').innerHTML = (result.jobs || []).map(job => `<tr class="job-row" data-job="${escapeHtml(job.id)}"><td class="job-id">${escapeHtml(job.id.slice(0, 12))}…</td><td>${escapeHtml(job.type)}</td><td><span class="status ${job.status}">${job.status}</span></td><td>${job.attempts}</td><td>${time(job.created_at)}</td><td>${job.status === 'FAILED' ? `<button class="retry" data-retry="${job.id}">Retry ↗</button>` : ''}</td></tr>`).join('');
    $('#empty').hidden = result.jobs?.length > 0;
  } catch (error) { showToast(error.message); }
}

$('#job-form').addEventListener('submit', async event => { event.preventDefault(); try { const payload = JSON.parse($('#job-payload').value); await api('/api/jobs', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ type: $('#job-type').value, payload }) }); showToast('Job queued'); event.target.reset(); $('#job-type').value = 'image.process'; $('#job-payload').value = '{"file":"demo.jpg"}'; refresh(); } catch (error) { showToast(error.message); } });
async function showDetails(jobID) {
  const modal = $('#job-detail');
  $('#detail-job-id').textContent = jobID;
  $('#timeline').innerHTML = '<p class="detail-loading">Loading events…</p>';
  modal.hidden = false;
  try {
    const result = await api(`/api/jobs/${jobID}/events`);
    $('#timeline').innerHTML = (result.events || []).map(event => `<article class="timeline-item"><div class="timeline-dot"></div><div class="timeline-content"><div class="timeline-head"><strong>${escapeHtml(event.event_type)}</strong><time>${time(event.occurred_at)}</time></div><p>attempt ${event.attempts}${event.error ? ` · ${escapeHtml(event.error)}` : ''}</p><pre>${escapeHtml(payload(event.payload))}</pre></div></article>`).join('') || '<p class="detail-loading">No events recorded yet.</p>';
  } catch (error) { $('#timeline').innerHTML = `<p class="detail-loading">${escapeHtml(error.message)}</p>`; }
}
function closeDetails() { $('#job-detail').hidden = true; }
document.addEventListener('click', async event => {
  const retry = event.target.closest('[data-retry]');
  if (retry) { try { await api(`/api/jobs/${retry.dataset.retry}/retry`, { method: 'POST' }); showToast('Job requeued'); refresh(); } catch (error) { showToast(error.message); } return; }
  const filter = event.target.closest('[data-status]');
  if (filter) { statusFilter = filter.dataset.status; document.querySelectorAll('.filter').forEach(item => item.classList.toggle('active', item.dataset.status === statusFilter)); refresh(); return; }
  const row = event.target.closest('[data-job]');
  if (row) showDetails(row.dataset.job);
  if (event.target.closest('[data-close-detail]')) closeDetails();
});
document.addEventListener('keydown', event => { if (event.key === 'Escape') closeDetails(); });
refresh(); setInterval(refresh, 4000);
