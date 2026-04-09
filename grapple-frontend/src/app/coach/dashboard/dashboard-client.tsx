'use client';

import { Badge, Spinner } from 'react-bootstrap';
import {
  ResponsiveContainer,
  LineChart,
  Line,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
} from 'recharts';
import { useGetDashboard } from '@/hook/dashboard';
import styles from './Dashboard.module.css';

function formatCurrency(cents: number) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  }).format(cents / 100);
}

function formatDate(iso: string) {
  return new Date(iso + 'T00:00:00').toLocaleDateString('en-US', {
    month: 'short', day: 'numeric',
  });
}

// Shorten month label: "2025-04" → "Apr"
function shortMonth(m: string) {
  const [year, month] = m.split('-');
  return new Date(Number(year), Number(month) - 1, 1)
    .toLocaleString('en-US', { month: 'short' });
}

// Shorten week label: "2025-W14" → "W14"
function shortWeek(w: string) {
  return w.split('-')[1]; // "W14"
}

export default function DashboardClient() {
  const { data, isPending } = useGetDashboard();

  if (isPending) {
    return (
      <div className={styles.loadingWrapper}>
        <Spinner animation="border" variant="dark" />
      </div>
    );
  }

  const d = data!;

  const revenueData = (d.revenue_by_month ?? []).map(r => ({
    month: shortMonth(r.month),
    revenue: r.revenue / 100,
  }));

  const attendanceData = (d.attendance_by_week ?? []).map(w => ({
    week: shortWeek(w.week),
    count: w.count,
  }));

  const hasAttention =
    (d.overdue_list?.length ?? 0) > 0 ||
    d.pending_requests > 0 ||
    (d.upcoming_renewals?.length ?? 0) > 0;

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h4 className={styles.title}>Dashboard</h4>
        <p className={styles.subtitle}>Your gym at a glance.</p>
      </div>

      {/* ── Metric Cards ── */}
      <div className={styles.cards}>
        <div className={styles.card}>
          <span className={styles.cardLabel}>Active Members</span>
          <span className={styles.cardValue}>{d.active_members}</span>
          <span className={styles.cardSub}>with active billing</span>
        </div>
        <div className={styles.card}>
          <span className={styles.cardLabel}>Monthly Revenue</span>
          <span className={styles.cardValue}>{formatCurrency(d.monthly_revenue)}</span>
          <span className={styles.cardSub}>paid this month</span>
        </div>
        <div className={styles.card}>
          <span className={styles.cardLabel}>Today's Attendance</span>
          <span className={styles.cardValue}>{d.today_attendance}</span>
          <span className={styles.cardSub}>check-ins today</span>
        </div>
        <div className={styles.card}>
          <span className={styles.cardLabel}>Overdue Members</span>
          <span className={`${styles.cardValue} ${d.overdue_count > 0 ? styles.cardValueDanger : ''}`}>
            {d.overdue_count}
          </span>
          <span className={styles.cardSub}>with overdue payments</span>
        </div>
      </div>

      {/* ── Charts ── */}
      <div className={styles.charts}>
        <div className={styles.chartCard}>
          <div className={styles.chartTitle}>Revenue — last 12 months</div>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={revenueData} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis
                dataKey="month"
                tick={{ fontSize: 11, fill: '#888' }}
                axisLine={false}
                tickLine={false}
              />
              <YAxis
                tick={{ fontSize: 11, fill: '#888' }}
                axisLine={false}
                tickLine={false}
                tickFormatter={v => `$${v}`}
                width={48}
              />
              <Tooltip
                formatter={(v) => [`$${Number(v).toFixed(2)}`, 'Revenue']}
                contentStyle={{ fontSize: 12, borderRadius: 6 }}
              />
              <Line
                type="monotone"
                dataKey="revenue"
                stroke="#111"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4 }}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div className={styles.chartCard}>
          <div className={styles.chartTitle}>Attendance — last 12 weeks</div>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={attendanceData} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" vertical={false} />
              <XAxis
                dataKey="week"
                tick={{ fontSize: 11, fill: '#888' }}
                axisLine={false}
                tickLine={false}
              />
              <YAxis
                tick={{ fontSize: 11, fill: '#888' }}
                axisLine={false}
                tickLine={false}
                allowDecimals={false}
                width={32}
              />
              <Tooltip
                formatter={(v) => [Number(v), 'Check-ins']}
                contentStyle={{ fontSize: 12, borderRadius: 6 }}
              />
              <Bar dataKey="count" fill="#111" radius={[3, 3, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* ── Needs Attention ── */}
      <div className={styles.attentionPanel}>
        <div className={styles.attentionHeader}>
          <p className={styles.attentionTitle}>Needs Attention</p>
          {hasAttention && (
            <Badge bg="danger" style={{ fontSize: 11 }}>
              {(d.overdue_list?.length ?? 0) +
                (d.pending_requests > 0 ? 1 : 0) +
                (d.upcoming_renewals?.length ?? 0)}
            </Badge>
          )}
        </div>

        {!hasAttention ? (
          <div className={styles.allClear}>
            <span className={styles.allClearIcon}>✓</span>
            <span>All caught up — nothing needs attention right now.</span>
          </div>
        ) : (
          <>
            {/* Overdue payments */}
            {(d.overdue_list?.length ?? 0) > 0 && (
              <div className={styles.attentionSection}>
                <div className={styles.attentionSectionLabel}>
                  Overdue Payments ({d.overdue_list.length})
                </div>
                {d.overdue_list.map(m => (
                  <div key={m.member_id} className={styles.attentionRow}>
                    <div>
                      <div className={styles.attentionName}>{m.member_name}</div>
                      <div className={styles.attentionMeta}>Due {formatDate(m.due_date)}</div>
                    </div>
                    <div className={styles.attentionRight}>
                      <span style={{ fontSize: 13, fontWeight: 600, color: '#dc3545' }}>
                        {formatCurrency(m.amount)}
                      </span>
                      <Badge bg="danger" style={{ fontSize: 11 }}>Overdue</Badge>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* Pending requests */}
            {d.pending_requests > 0 && (
              <div className={styles.attentionSection}>
                <div className={styles.attentionSectionLabel}>Pending Join Requests</div>
                <div className={styles.attentionRow}>
                  <div>
                    <div className={styles.attentionName}>
                      {d.pending_requests} member{d.pending_requests !== 1 ? 's' : ''} waiting for approval
                    </div>
                  </div>
                  <div className={styles.attentionRight}>
                    <a href="/coach/user" className={styles.pendingLink}>
                      Review →
                    </a>
                  </div>
                </div>
              </div>
            )}

            {/* Upcoming renewals */}
            {(d.upcoming_renewals?.length ?? 0) > 0 && (
              <div className={styles.attentionSection}>
                <div className={styles.attentionSectionLabel}>
                  Renewals Due This Week ({d.upcoming_renewals.length})
                </div>
                {d.upcoming_renewals.map((r, i) => (
                  <div key={`${r.member_id}-${i}`} className={styles.attentionRow}>
                    <div>
                      <div className={styles.attentionName}>{r.member_name}</div>
                      <div className={styles.attentionMeta}>
                        {r.plan_name} · Due {formatDate(r.due_date)}
                      </div>
                    </div>
                    <div className={styles.attentionRight}>
                      <span style={{ fontSize: 13, fontWeight: 600 }}>
                        {formatCurrency(r.amount)}
                      </span>
                      <Badge bg="warning" text="dark" style={{ fontSize: 11 }}>Unpaid</Badge>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
