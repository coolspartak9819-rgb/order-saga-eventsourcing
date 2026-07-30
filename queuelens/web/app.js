const api = async (path, options) => { const response = await fetch(path, options); const body = await response.json(); if (!response.ok) throw new Error(body.error || 'Request failed'); return body; };
const $ = selector => document.querySelector(selector);
let statusFilter = '';

function escapeHtml(value) { return String(value ?? '').replace(/[&<>"']/g, ch => ({ '&':'&amp;', '<':'&lt;', '>':'&gt;', '"':'&quot;', "'":'&#039;' }[ch])); }
function showToast(message) { const toast = $('#toast'); toast.textContent = message; toast.classList.add('visible'); setTimeout(() => toast.classList.remove('visible'), 2200); }
function time(value) { return new Date(value).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }); }

async function refresh() {
  try {
    const [stats, result] = await Promise.all([api('/api/stats'), api(`/api/jobs${statusFilter ? `?status=${statusFilter}` : ''}`)]);
    const statuses = ['PENDING', 'RUNNING', 'COMPLETED', 'FAILED'];
    $('#stats').innerHTML = statuses.map(status => `<div class="stat"><label>${status.toLowerCase()}</label><strong>${stats[status] || 0}</strong></div>`).join('');
    $('#jobs').innerHTML = (result.jobs || []).map(job => `<tr><td class="job-id">${escapeHtml(job.id.slice(0, 12))}…</td><td>${escapeHtml(job.type)}</td><td><span class="status ${job.status}">${job.status}</span></td><td>${job.attempts}</td><td>${time(job.created_at)}</td><td>${job.status === 'FAILED' ? `<button class="retry" data-retry="${job.id}">Retry ↗</button>` : ''}</td></tr>`).join('');
    $('#empty').hidden = result.jobs?.length > 0;
  } catch (error) { showToast(error.message); }
}

$('#job-form').addEventListener('submit', async event => { event.preventDefault(); try { const payload = JSON.parse($('#job-payload').value); await api('/api/jobs', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ type: $('#job-type').value, payload }) }); showToast('Job queued'); event.target.reset(); $('#job-type').value = 'image.process'; $('#job-payload').value = '{"file":"demo.jpg"}'; refresh(); } catch (error) { showToast(error.message); } });
document.addEventListener('click', async event => { const button = event.target.closest('[data-status], [data-retry]'); if (!button) return; if (button.dataset.status !== undefined) { statusFilter = button.dataset.status; document.querySelectorAll('.filter').forEach(item => item.classList.toggle('active', item.dataset.status === statusFilter)); refresh(); } if (button.dataset.retry) { try { await api(`/api/jobs/${button.dataset.retry}/retry`, { method: 'POST' }); showToast('Job requeued'); refresh(); } catch (error) { showToast(error.message); } } });
refresh(); setInterval(refresh, 4000);
