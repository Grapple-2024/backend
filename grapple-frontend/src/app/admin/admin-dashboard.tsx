'use client';

import React, { useState, useCallback } from 'react';
import { Spinner, Dropdown } from 'react-bootstrap';
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
} from 'recharts';
import styles from './Admin.module.css';
import {
  useAdminMetrics,
  useAdminGyms,
  useAdminGymDetail,
  useAdminUpdateGym,
  useAdminDeleteGym,
} from '@/hook/admin';
import type { AdminGym } from '@/api-requests/admin';

// ── Helpers ──────────────────────────────────────────────────────────────────

function fmt$(cents: number) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency', currency: 'USD', maximumFractionDigits: 0,
  }).format(cents / 100);
}

function shortMonth(m: string) {
  const [year, month] = m.split('-');
  return new Date(Number(year), Number(month) - 1, 1)
    .toLocaleString('en-US', { month: 'short' });
}

function relativeTime(iso: string | null) {
  if (!iso) return '—';
  const diff = Date.now() - new Date(iso).getTime();
  const days = Math.floor(diff / 86400000);
  if (days === 0) return 'today';
  if (days === 1) return 'yesterday';
  if (days < 30) return `${days}d ago`;
  return `${Math.floor(days / 30)}mo ago`;
}

function fmtDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-US', {
    month: 'short', day: 'numeric', year: 'numeric',
  });
}

// ── Tier badge ────────────────────────────────────────────────────────────────

function TierBadge({ tier }: { tier: number }) {
  const cls = tier === 3 ? styles.tier3 : tier === 2 ? styles.tier2 : styles.tier1;
  return <span className={`${styles.tierBadge} ${cls}`}>T{tier}</span>;
}

// ── Delete modal ──────────────────────────────────────────────────────────────

function DeleteModal({
  gym,
  onCancel,
  onDeleted,
}: {
  gym: AdminGym;
  onCancel: () => void;
  onDeleted: () => void;
}) {
  const [confirm, setConfirm] = useState('');
  const [password, setPassword] = useState('');
  const deleteMutation = useAdminDeleteGym();

  const ready = confirm === 'DELETE' && password.length > 0;

  const handleDelete = () => {
    deleteMutation.mutate(
      { id: gym.id, password },
      {
        onSuccess: () => { onDeleted(); },
        onError: (err: any) => {
          alert(err?.response?.data?.error?.[0] ?? 'Delete failed');
        },
      },
    );
  };

  return (
    <div className={styles.modalOverlay}>
      <div className={styles.modal}>
        <div className={styles.modalTitle}>Force Delete Gym</div>
        <div className={styles.modalSub}>{gym.name}</div>
        <div className={styles.warningBox}>
          This permanently deletes the gym, all members, billing records, attendance,
          belt promotions, and payment history. This cannot be undone.
        </div>
        <input
          className={styles.modalInput}
          placeholder='Type DELETE to confirm'
          value={confirm}
          onChange={e => setConfirm(e.target.value)}
          autoFocus
        />
        <input
          className={styles.modalInput}
          type="password"
          placeholder="Admin password"
          value={password}
          onChange={e => setPassword(e.target.value)}
        />
        <div className={styles.modalActions}>
          <button className={styles.btnCancel} onClick={onCancel}>Cancel</button>
          <button
            className={styles.btnDanger}
            disabled={!ready || deleteMutation.isPending}
            onClick={handleDelete}
          >
            {deleteMutation.isPending ? 'Deleting…' : 'Delete Forever'}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Tier modal ────────────────────────────────────────────────────────────────

function TierModal({
  gym,
  onClose,
}: {
  gym: AdminGym;
  onClose: () => void;
}) {
  const [tier, setTier] = useState(gym.tier);
  const updateGym = useAdminUpdateGym();

  return (
    <div className={styles.modalOverlay}>
      <div className={styles.modal}>
        <div className={styles.modalTitle}>Update Tier — {gym.name}</div>
        <div className={styles.modalSub}>Current tier: T{gym.tier}</div>
        <select
          className={styles.tierSelect}
          value={tier}
          onChange={e => setTier(Number(e.target.value))}
        >
          <option value={1}>Tier 1 — Free</option>
          <option value={2}>Tier 2 — Pro</option>
          <option value={3}>Tier 3 — Enterprise</option>
        </select>
        <div className={styles.modalActions}>
          <button className={styles.btnCancel} onClick={onClose}>Cancel</button>
          <button
            className={styles.btnPrimary}
            disabled={tier === gym.tier || updateGym.isPending}
            onClick={() =>
              updateGym.mutate(
                { id: gym.id, action: 'update_tier', payload: { tier } },
                { onSuccess: onClose },
              )
            }
          >
            {updateGym.isPending ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Note modal ────────────────────────────────────────────────────────────────

function NoteModal({ gym, onClose }: { gym: AdminGym; onClose: () => void }) {
  const [note, setNote] = useState('');
  const updateGym = useAdminUpdateGym();

  return (
    <div className={styles.modalOverlay}>
      <div className={styles.modal}>
        <div className={styles.modalTitle}>Add Note — {gym.name}</div>
        <textarea
          className={styles.modalInput}
          rows={4}
          placeholder="Enter admin note…"
          value={note}
          onChange={e => setNote(e.target.value)}
          style={{ resize: 'vertical' }}
          autoFocus
        />
        <div className={styles.modalActions}>
          <button className={styles.btnCancel} onClick={onClose}>Cancel</button>
          <button
            className={styles.btnPrimary}
            disabled={!note.trim() || updateGym.isPending}
            onClick={() =>
              updateGym.mutate(
                { id: gym.id, action: 'add_note', payload: { note } },
                { onSuccess: onClose },
              )
            }
          >
            {updateGym.isPending ? 'Saving…' : 'Save Note'}
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Gym detail drawer ─────────────────────────────────────────────────────────

function GymDrawer({ gymId, onClose }: { gymId: string; onClose: () => void }) {
  const { data, isPending } = useAdminGymDetail(gymId);

  return (
    <>
      <div className={styles.drawerOverlay} onClick={onClose} />
      <div className={styles.drawer}>
        <div className={styles.drawerHeader}>
          <div className={styles.drawerTitle}>
            {isPending ? 'Loading…' : data?.gym?.name ?? 'Gym Detail'}
          </div>
          <button className={styles.drawerClose} onClick={onClose}>✕</button>
        </div>

        {isPending ? (
          <div className={styles.loadingWrap}><Spinner animation="border" variant="light" /></div>
        ) : !data ? null : (
          <>
            {/* Key stats */}
            <div className={styles.drawerSection}>
              <div className={styles.drawerSectionTitle}>Overview</div>
              <div className={styles.drawerRow}>
                <span className={styles.drawerRowKey}>Members</span>
                <span className={styles.drawerRowVal}>{data.member_count}</span>
              </div>
              <div className={styles.drawerRow}>
                <span className={styles.drawerRowKey}>Revenue (30d)</span>
                <span className={styles.drawerRowVal}>{fmt$(data.revenue_30d)}</span>
              </div>
              <div className={styles.drawerRow}>
                <span className={styles.drawerRowKey}>Address</span>
                <span className={styles.drawerRowVal}>
                  {[data.gym.address_line_1, data.gym.city, data.gym.state, data.gym.zip]
                    .filter(Boolean).join(', ')}
                </span>
              </div>
              <div className={styles.drawerRow}>
                <span className={styles.drawerRowKey}>Owner Email</span>
                <span className={styles.drawerRowVal}>{data.gym.coach_email ?? '—'}</span>
              </div>
              <div className={styles.drawerRow}>
                <span className={styles.drawerRowKey}>Created</span>
                <span className={styles.drawerRowVal}>
                  {data.gym.created_at ? fmtDate(data.gym.created_at) : '—'}
                </span>
              </div>
              <div className={styles.drawerRow}>
                <span className={styles.drawerRowKey}>Tier</span>
                <span className={styles.drawerRowVal}>
                  <TierBadge tier={data.gym.tier ?? 1} />
                </span>
              </div>
            </div>

            {/* Admin notes */}
            <div className={styles.drawerSection}>
              <div className={styles.drawerSectionTitle}>Admin Notes</div>
              {data.admin_notes.length === 0 ? (
                <div className={styles.empty}>No notes yet.</div>
              ) : data.admin_notes.map(n => (
                <div key={n.id} className={styles.noteEntry}>
                  <div>{n.metadata?.note}</div>
                  <div className={styles.logMeta}>{fmtDate(n.timestamp)}</div>
                </div>
              ))}
            </div>

            {/* Activity log */}
            <div className={styles.drawerSection}>
              <div className={styles.drawerSectionTitle}>Activity Log</div>
              {data.activity_log.length === 0 ? (
                <div className={styles.empty}>No activity logged.</div>
              ) : data.activity_log.map(l => (
                <div key={l.id} className={styles.logEntry}>
                  <div className={styles.logAction}>{l.action.replace(/_/g, ' ')}</div>
                  {l.metadata && Object.keys(l.metadata).length > 0 && (
                    <div className={styles.logMeta}>
                      {JSON.stringify(l.metadata)}
                    </div>
                  )}
                  <div className={styles.logMeta}>{fmtDate(l.timestamp)}</div>
                </div>
              ))}
            </div>
          </>
        )}
      </div>
    </>
  );
}

// ── Main dashboard ────────────────────────────────────────────────────────────

export default function AdminDashboard() {
  const [search, setSearch] = useState('');
  const [searchInput, setSearchInput] = useState('');
  const [page, setPage] = useState(1);

  const [drawerGymId, setDrawerGymId] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AdminGym | null>(null);
  const [tierTarget, setTierTarget] = useState<AdminGym | null>(null);
  const [noteTarget, setNoteTarget] = useState<AdminGym | null>(null);

  const metrics = useAdminMetrics();
  const roster = useAdminGyms(search, page);

  const handleSearch = useCallback(() => {
    setSearch(searchInput);
    setPage(1);
  }, [searchInput]);

  const mrrData = (metrics.data?.mrr_by_month ?? []).map(m => ({
    month: shortMonth(m.month),
    mrr: m.mrr / 100,
  }));

  const pageSize = 25;
  const totalPages = Math.ceil((roster.data?.total_count ?? 0) / pageSize);

  return (
    <div className={styles.shell}>
      {/* Header */}
      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <span className={styles.logoText}>Grapple</span>
          <span className={styles.adminBadge}>Admin</span>
        </div>
      </div>

      <div className={styles.content}>

        {/* ── Metrics ── */}
        {metrics.isPending ? (
          <div className={styles.loadingWrap}><Spinner animation="border" variant="light" /></div>
        ) : metrics.data && (
          <>
            <div className={styles.metricsGrid}>
              <div className={styles.metricCard}>
                <span className={styles.metricLabel}>Total MRR</span>
                <span className={styles.metricValue}>{fmt$(metrics.data.total_mrr)}</span>
                <span className={styles.metricSub}>this month</span>
              </div>
              <div className={styles.metricCard}>
                <span className={styles.metricLabel}>Active Gyms</span>
                <span className={styles.metricValue}>{metrics.data.active_gyms}</span>
                <span className={styles.metricSub}>+{metrics.data.new_gyms_this_month} this month</span>
              </div>
              <div className={styles.metricCard}>
                <span className={styles.metricLabel}>Total Students</span>
                <span className={styles.metricValue}>{metrics.data.total_students.toLocaleString()}</span>
                <span className={styles.metricSub}>across all gyms</span>
              </div>
              <div className={styles.metricCard}>
                <span className={styles.metricLabel}>Churn Rate</span>
                <span className={styles.metricValue}>{metrics.data.churn_rate.toFixed(1)}%</span>
                <span className={styles.metricSub}>last 30 days</span>
              </div>
              <div className={styles.metricCard}>
                <span className={styles.metricLabel}>Avg Students</span>
                <span className={styles.metricValue}>{metrics.data.avg_students_per_gym.toFixed(1)}</span>
                <span className={styles.metricSub}>per gym</span>
              </div>
              <div className={styles.metricCard}>
                <span className={styles.metricLabel}>New Gyms</span>
                <span className={styles.metricValue}>{metrics.data.new_gyms_this_month}</span>
                <span className={styles.metricSub}>this month</span>
              </div>
            </div>

            {/* Feature adoption */}
            <div className={styles.adoptionRow}>
              {([
                { label: 'Billing Adoption', pct: metrics.data.feature_adoption.billing_pct },
                { label: 'Attendance Adoption', pct: metrics.data.feature_adoption.attendance_pct },
                { label: 'Belt Tracking Adoption', pct: metrics.data.feature_adoption.belt_tracking_pct },
              ] as { label: string; pct: number }[]).map(f => (
                <div key={f.label} className={styles.adoptionCard}>
                  <div className={styles.adoptionLabel}>{f.label}</div>
                  <div className={styles.adoptionBar}>
                    <div
                      className={styles.adoptionFill}
                      style={{ width: `${Math.min(f.pct, 100)}%` }}
                    />
                  </div>
                  <div className={styles.adoptionPct}>{f.pct.toFixed(0)}%</div>
                </div>
              ))}
            </div>

            {/* MRR trend */}
            <div className={styles.chartSection}>
              <div className={styles.chartTitle}>MRR — last 12 months</div>
              <ResponsiveContainer width="100%" height={200}>
                <LineChart data={mrrData} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#2a2a2a" />
                  <XAxis dataKey="month" tick={{ fontSize: 11, fill: '#666' }} axisLine={false} tickLine={false} />
                  <YAxis tick={{ fontSize: 11, fill: '#666' }} axisLine={false} tickLine={false}
                    tickFormatter={v => `$${v}`} width={56} />
                  <Tooltip
                    formatter={(v) => [`$${Number(v).toFixed(2)}`, 'MRR']}
                    contentStyle={{ background: '#1a1a1a', border: '1px solid #333', borderRadius: 6, fontSize: 12 }}
                    labelStyle={{ color: '#aaa' }}
                  />
                  <Line type="monotone" dataKey="mrr" stroke="#4ade80" strokeWidth={2} dot={false} activeDot={{ r: 4 }} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </>
        )}

        {/* ── Gym Roster ── */}
        <div className={styles.tableSection}>
          <div className={styles.tableHeader}>
            <div className={styles.sectionTitle} style={{ margin: 0 }}>
              Gym Roster
              {roster.data && (
                <span style={{ color: '#555', fontWeight: 400, marginLeft: 8 }}>
                  ({roster.data.total_count})
                </span>
              )}
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <input
                className={styles.searchInput}
                placeholder="Search gyms…"
                value={searchInput}
                onChange={e => setSearchInput(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && handleSearch()}
              />
              <button className={styles.pageBtn} onClick={handleSearch}>Search</button>
              <button className={styles.pageBtn} onClick={() => { setSearchInput(''); setSearch(''); setPage(1); }}>Clear</button>
            </div>
          </div>

          {roster.isPending ? (
            <div className={styles.loadingWrap}><Spinner animation="border" variant="light" /></div>
          ) : (
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Gym</th>
                  <th>Owner</th>
                  <th>Address</th>
                  <th>Students</th>
                  <th>Tier</th>
                  <th>Billing</th>
                  <th>Last Activity</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {(roster.data?.data ?? []).map(gym => (
                  <tr key={gym.id}>
                    <td>
                      <div
                        className={styles.gymName}
                        onClick={() => setDrawerGymId(gym.id)}
                      >
                        {gym.name}
                      </div>
                      <div className={styles.addressLine}>{fmtDate(gym.created_at)}</div>
                    </td>
                    <td>
                      <div>{gym.owner_name}</div>
                      <div className={styles.ownerLine}>{gym.owner_email}</div>
                    </td>
                    <td>
                      <div>{gym.address}</div>
                    </td>
                    <td>{gym.student_count}</td>
                    <td><TierBadge tier={gym.tier} /></td>
                    <td>
                      <span className={`${styles.billingDot} ${gym.has_billing ? styles.billingOn : styles.billingOff}`} />
                      {gym.has_billing ? 'Active' : 'None'}
                    </td>
                    <td className={styles.activityDot}>{relativeTime(gym.last_activity)}</td>
                    <td>
                      <Dropdown>
                        <Dropdown.Toggle as="button" className={styles.actionsBtn} id={`actions-${gym.id}`}>
                          Actions ▾
                        </Dropdown.Toggle>
                        <Dropdown.Menu
                          style={{ background: '#1a1a1a', border: '1px solid #333', minWidth: 180 }}
                        >
                          <Dropdown.Item
                            style={{ color: '#ddd', fontSize: 13 }}
                            onClick={() => setDrawerGymId(gym.id)}
                          >
                            View Detail
                          </Dropdown.Item>
                          <Dropdown.Item
                            style={{ color: '#ddd', fontSize: 13 }}
                            href={`/coach/my-gym`}
                            target="_blank"
                          >
                            View as Coach ↗
                          </Dropdown.Item>
                          <Dropdown.Divider style={{ borderColor: '#333' }} />
                          <Dropdown.Item
                            style={{ color: '#ddd', fontSize: 13 }}
                            onClick={() => setTierTarget(gym)}
                          >
                            Upgrade / Downgrade Tier
                          </Dropdown.Item>
                          <Dropdown.Item
                            style={{ color: '#ddd', fontSize: 13 }}
                            onClick={() => setNoteTarget(gym)}
                          >
                            Add Note
                          </Dropdown.Item>
                          <Dropdown.Divider style={{ borderColor: '#333' }} />
                          <Dropdown.Item
                            style={{ color: '#ef4444', fontSize: 13 }}
                            onClick={() => setDeleteTarget(gym)}
                          >
                            Force Delete…
                          </Dropdown.Item>
                        </Dropdown.Menu>
                      </Dropdown>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className={styles.pagination}>
              <button className={styles.pageBtn} onClick={() => setPage(1)} disabled={page === 1}>«</button>
              <button className={styles.pageBtn} onClick={() => setPage(p => p - 1)} disabled={page === 1}>‹</button>
              <span>Page {page} of {totalPages}</span>
              <button className={styles.pageBtn} onClick={() => setPage(p => p + 1)} disabled={page >= totalPages}>›</button>
              <button className={styles.pageBtn} onClick={() => setPage(totalPages)} disabled={page >= totalPages}>»</button>
            </div>
          )}
        </div>
      </div>

      {/* Modals */}
      {deleteTarget && (
        <DeleteModal
          gym={deleteTarget}
          onCancel={() => setDeleteTarget(null)}
          onDeleted={() => setDeleteTarget(null)}
        />
      )}
      {tierTarget && (
        <TierModal gym={tierTarget} onClose={() => setTierTarget(null)} />
      )}
      {noteTarget && (
        <NoteModal gym={noteTarget} onClose={() => setNoteTarget(null)} />
      )}

      {/* Drawer */}
      {drawerGymId && (
        <GymDrawer gymId={drawerGymId} onClose={() => setDrawerGymId(null)} />
      )}
    </div>
  );
}
