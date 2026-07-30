import React, { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { ArrowRight, CalendarDays, ChevronLeft, CirclePlus, ClipboardList, MapPin, Package, Search, Truck, X } from 'lucide-react';
import './styles.css';

const apiBase = import.meta.env.VITE_API_URL || '';
const emptyForm = { senderCity: '', senderAddress: '', recipientCity: '', recipientAddress: '', weightKg: '', pickupDate: '' };

async function api(path, options) {
  const response = await fetch(`${apiBase}${path}`, { headers: { 'Content-Type': 'application/json', ...(options?.headers || {}) }, ...options });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(Object.values(body.errors || {}).flat()[0] || body.message || 'Request failed');
  }
  return response.status === 204 ? null : response.json();
}

function App() {
  const [orders, setOrders] = useState([]);
  const [selected, setSelected] = useState(null);
  const [formOpen, setFormOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const loadOrders = async () => {
    try {
      setLoading(true);
      setOrders(await api(`/api/orders?search=${encodeURIComponent(search)}${status ? `&status=${status}` : ''}`));
      setError('');
    } catch (err) { setError(err.message); } finally { setLoading(false); }
  };

  useEffect(() => { loadOrders(); }, [search, status]);

  const stats = useMemo(() => ({ total: orders.length, today: orders.filter(order => order.pickupDate === new Date().toISOString().slice(0, 10)).length, weight: orders.reduce((sum, order) => sum + order.weightKg, 0) }), [orders]);

  const openOrder = async (id) => {
    try { setSelected(await api(`/api/orders/${id}`)); setError(''); } catch (err) { setError(err.message); }
  };

  return <div className="app-shell">
    <header className="topbar">
      <div className="brand"><div className="brand-mark"><Truck size={19} /></div><div><strong>ParcelPilot</strong><span>delivery desk</span></div></div>
      <div className="topbar-meta"><span className="live-dot" /> Operations online <button className="avatar">IS</button></div>
    </header>
    <main className="workspace">
      <section className="page-heading"><div><p className="eyebrow">Orders / Dispatch board</p><h1>Delivery orders</h1><p className="subtitle">Plan pickups and keep every shipment visible from one place.</p></div><button className="primary-button" onClick={() => setFormOpen(true)}><CirclePlus size={18} /> New order</button></section>
      <section className="stats-grid"><Stat label="Active orders" value={stats.total} icon={<ClipboardList />} accent="blue" /><Stat label="Pickups today" value={stats.today} icon={<CalendarDays />} accent="violet" /><Stat label="Total weight" value={`${stats.weight.toFixed(1)} kg`} icon={<Package />} accent="orange" /></section>
      <section className="orders-section">
        <div className="section-toolbar"><div><h2>All orders</h2><span className="muted">{orders.length} shipments in your workspace</span></div><div className="filters"><label className="search-box"><Search size={16} /><input value={search} onChange={event => setSearch(event.target.value)} placeholder="Search number or city" /></label><select value={status} onChange={event => setStatus(event.target.value)}><option value="">All statuses</option><option value="Created">Created</option><option value="Accepted">Accepted</option><option value="InTransit">In transit</option><option value="Delivered">Delivered</option><option value="Cancelled">Cancelled</option></select></div></div>
        {error && <div className="error-banner">{error}</div>}
        {loading ? <div className="empty-state">Loading orders...</div> : orders.length === 0 ? <div className="empty-state"><Package size={30} /><strong>No orders yet</strong><span>Create the first delivery to see it here.</span><button className="secondary-button" onClick={() => setFormOpen(true)}>Create order</button></div> : <div className="table-wrap"><table><thead><tr><th>Order</th><th>Route</th><th>Pickup</th><th>Weight</th><th>Status</th><th /></tr></thead><tbody>{orders.map(order => <tr key={order.id} onClick={() => openOrder(order.id)}><td><strong className="order-number">{order.orderNumber}</strong><small>{formatDate(order.createdAt)}</small></td><td><div className="route-cell"><span>{order.senderCity}</span><ArrowRight size={14} /><span>{order.recipientCity}</span></div></td><td>{formatDateOnly(order.pickupDate)}</td><td>{order.weightKg} kg</td><td><Status status={order.status} /></td><td><ChevronLeft className="row-chevron" size={17} /></td></tr>)}</tbody></table></div>}
      </section>
    </main>
    {formOpen && <CreateOrder onClose={() => setFormOpen(false)} onCreated={async order => { setFormOpen(false); await loadOrders(); setSelected(order); }} />}
    {selected && <OrderDetail order={selected} onClose={() => setSelected(null)} />}
  </div>;
}

function Stat({ label, value, icon, accent }) { return <div className="stat-card"><div className={`stat-icon ${accent}`}>{icon}</div><div><span>{label}</span><strong>{value}</strong></div></div>; }
function Status({ status }) { return <span className={`status status-${status.toLowerCase()}`}><i />{status === 'InTransit' ? 'In transit' : status}</span>; }
function formatDate(value) { return new Intl.DateTimeFormat('en', { day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(value)); }
function formatDateOnly(value) { return new Intl.DateTimeFormat('en', { day: '2-digit', month: 'short' }).format(new Date(`${value}T00:00:00`)); }

function CreateOrder({ onClose, onCreated }) {
  const [form, setForm] = useState({ ...emptyForm, pickupDate: new Date().toISOString().slice(0, 10) });
  const [saving, setSaving] = useState(false); const [error, setError] = useState('');
  const update = event => setForm({ ...form, [event.target.name]: event.target.value });
  const submit = async event => { event.preventDefault(); setSaving(true); setError(''); try { await onCreated(await api('/api/orders', { method: 'POST', body: JSON.stringify({ ...form, weightKg: Number(form.weightKg) }) })); } catch (err) { setError(err.message); } finally { setSaving(false); } };
  return <div className="overlay"><div className="drawer"><div className="drawer-header"><div><p className="eyebrow">New shipment</p><h2>Create an order</h2></div><button className="icon-button" onClick={onClose}><X size={19} /></button></div><form onSubmit={submit}><div className="form-section"><h3>Route</h3><div className="form-grid"><Field label="Sender city" name="senderCity" value={form.senderCity} onChange={update} placeholder="Saint Petersburg" /><Field label="Sender address" name="senderAddress" value={form.senderAddress} onChange={update} placeholder="Nevsky prospect, 28" /><Field label="Recipient city" name="recipientCity" value={form.recipientCity} onChange={update} placeholder="Moscow" /><Field label="Recipient address" name="recipientAddress" value={form.recipientAddress} onChange={update} placeholder="Tverskaya street, 12" /></div></div><div className="form-section"><h3>Shipment details</h3><div className="form-grid"><Field label="Weight, kg" name="weightKg" type="number" min="0.1" step="0.1" value={form.weightKg} onChange={update} placeholder="2.5" /><Field label="Pickup date" name="pickupDate" type="date" value={form.pickupDate} onChange={update} /></div></div>{error && <div className="form-error">{error}</div>}<div className="form-actions"><button type="button" className="secondary-button" onClick={onClose}>Cancel</button><button className="primary-button" disabled={saving}>{saving ? 'Creating...' : 'Create order'}</button></div></form></div></div>;
}
function Field({ label, ...props }) { return <label className="field"><span>{label}</span><input required {...props} /></label>; }

function OrderDetail({ order, onClose }) { return <div className="overlay"><aside className="detail-panel"><div className="drawer-header"><div><button className="back-button" onClick={onClose}><ChevronLeft size={17} /> Back to orders</button><h2>{order.orderNumber}</h2><Status status={order.status} /></div><button className="icon-button" onClick={onClose}><X size={19} /></button></div><div className="route-preview"><div className="route-node"><span className="node-dot sender" /><small>FROM</small><strong>{order.senderCity}</strong><span>{order.senderAddress}</span></div><div className="route-line"><ArrowRight size={18} /></div><div className="route-node"><span className="node-dot recipient" /><small>TO</small><strong>{order.recipientCity}</strong><span>{order.recipientAddress}</span></div></div><div className="detail-grid"><Detail label="Pickup date" value={formatDateOnly(order.pickupDate)} /><Detail label="Cargo weight" value={`${order.weightKg} kg`} /><Detail label="Created" value={formatDate(order.createdAt)} /><Detail label="Order ID" value={order.id.slice(0, 8)} /></div><div className="timeline"><h3>Order timeline</h3><TimelineItem title="Order created" note="Shipment details received" active /><TimelineItem title="Pickup scheduled" note={formatDateOnly(order.pickupDate)} active={false} /></div></aside></div>; }
function Detail({ label, value }) { return <div><span>{label}</span><strong>{value}</strong></div>; }
function TimelineItem({ title, note, active }) { return <div className={`timeline-item ${active ? 'active' : ''}`}><i /><div><strong>{title}</strong><span>{note}</span></div></div>; }

createRoot(document.getElementById('root')).render(<App />);
