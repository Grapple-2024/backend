'use client';

import { useState } from 'react';
import { Badge, Button, Dropdown, Form, Spinner, Table } from 'react-bootstrap';
import { useGetPaymentRecords, useUpdatePaymentStatus, useUpdateBillingStatus } from '@/hook/memberBilling';
import { PaymentRecord } from '@/api-requests/memberBilling';
import styles from './Billing.module.css';

const STATUS_COLORS: Record<PaymentRecord['status'], string> = {
  paid: 'success',
  unpaid: 'warning',
  overdue: 'danger',
};

function formatPrice(cents: number, currency = 'usd') {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency.toUpperCase(),
  }).format(cents / 100);
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

export default function BillingClient() {
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [notesTarget, setNotesTarget] = useState<{ recordId: string; notes: string } | null>(null);

  const records = useGetPaymentRecords(undefined, statusFilter || undefined);
  const updatePayment = useUpdatePaymentStatus();
  const updateBilling = useUpdateBillingStatus();

  const handleMark = (recordId: string, status: 'paid' | 'unpaid' | 'overdue') => {
    updatePayment.mutate({ recordId, status });
  };

  const data = records.data ?? [];

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div>
          <h4 className={styles.title}>Member Billing</h4>
          <p className={styles.subtitle}>Track payments for all active members.</p>
        </div>
        <div className={styles.filters}>
          <Form.Select
            value={statusFilter}
            onChange={e => setStatusFilter(e.target.value)}
            className={styles.filterSelect}
          >
            <option value="">All statuses</option>
            <option value="unpaid">Unpaid</option>
            <option value="paid">Paid</option>
            <option value="overdue">Overdue</option>
          </Form.Select>
        </div>
      </div>

      {records.isPending ? (
        <div className={styles.loadingWrapper}>
          <Spinner animation="border" variant="dark" />
        </div>
      ) : data.length === 0 ? (
        <div className={styles.emptyState}>
          <p>No payment records yet. Assign plans to members from the{' '}
            <a href="/coach/user">User Management</a> page.
          </p>
        </div>
      ) : (
        <div className={styles.tableWrapper}>
          <Table className={styles.table} hover>
            <thead>
              <tr>
                <th>Member</th>
                <th>Plan</th>
                <th>Amount</th>
                <th>Due Date</th>
                <th>Status</th>
                <th>Paid On</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {data.map(record => (
                <tr key={record.id}>
                  <td className={styles.memberCell}>{record.member_name}</td>
                  <td>{record.plan_name}</td>
                  <td>{formatPrice(record.amount, record.currency)}</td>
                  <td>{formatDate(record.due_date)}</td>
                  <td>
                    <Badge bg={STATUS_COLORS[record.status]} className={styles.statusBadge}>
                      {record.status}
                    </Badge>
                  </td>
                  <td>{record.paid_at ? formatDate(record.paid_at) : '—'}</td>
                  <td>
                    <div className={styles.actions}>
                      {record.status !== 'paid' && (
                        <Button
                          size="sm"
                          variant="outline-success"
                          disabled={updatePayment.isPending}
                          onClick={() => handleMark(record.id!, 'paid')}
                        >
                          Mark Paid
                        </Button>
                      )}
                      {record.status !== 'overdue' && (
                        <Button
                          size="sm"
                          variant="outline-danger"
                          disabled={updatePayment.isPending}
                          onClick={() => handleMark(record.id!, 'overdue')}
                        >
                          Overdue
                        </Button>
                      )}
                      {record.status !== 'unpaid' && (
                        <Button
                          size="sm"
                          variant="outline-secondary"
                          disabled={updatePayment.isPending}
                          onClick={() => handleMark(record.id!, 'unpaid')}
                        >
                          Unpaid
                        </Button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        </div>
      )}

      <div className={styles.summary}>
        <div className={styles.summaryCard}>
          <span className={styles.summaryLabel}>Total Records</span>
          <span className={styles.summaryValue}>{data.length}</span>
        </div>
        <div className={styles.summaryCard}>
          <span className={styles.summaryLabel}>Paid</span>
          <span className={`${styles.summaryValue} ${styles.paid}`}>
            {data.filter(r => r.status === 'paid').length}
          </span>
        </div>
        <div className={styles.summaryCard}>
          <span className={styles.summaryLabel}>Unpaid</span>
          <span className={`${styles.summaryValue} ${styles.unpaid}`}>
            {data.filter(r => r.status === 'unpaid').length}
          </span>
        </div>
        <div className={styles.summaryCard}>
          <span className={styles.summaryLabel}>Overdue</span>
          <span className={`${styles.summaryValue} ${styles.overdue}`}>
            {data.filter(r => r.status === 'overdue').length}
          </span>
        </div>
        <div className={styles.summaryCard}>
          <span className={styles.summaryLabel}>Revenue (paid)</span>
          <span className={styles.summaryValue}>
            {formatPrice(data.filter(r => r.status === 'paid').reduce((sum, r) => sum + r.amount, 0))}
          </span>
        </div>
      </div>
    </div>
  );
}
