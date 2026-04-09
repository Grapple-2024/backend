import { Badge, Button, Spinner } from 'react-bootstrap';
import { User } from '@/components/UserTable';
import { useGetPaymentRecords, useUpdatePaymentStatus, useGetMemberBilling } from '@/hook/memberBilling';
import styles from '../MemberDrawer.module.css';

interface Props {
  member: User;
}

function formatPrice(cents: number) {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(cents / 100);
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

const STATUS_COLORS: Record<string, string> = {
  paid: 'success',
  unpaid: 'warning',
  overdue: 'danger',
};

export default function BillingTab({ member }: Props) {
  const billing = useGetMemberBilling(member.requestor_id);
  const payments = useGetPaymentRecords(member.requestor_id);
  const updatePayment = useUpdatePaymentStatus();

  if (billing.isPending || payments.isPending) {
    return <div className={styles.tabLoading}><Spinner size="sm" /></div>;
  }

  const activeBilling = (billing.data ?? []).find(b => b.status === 'active');
  const records = payments.data ?? [];

  return (
    <div className={styles.tabContent}>
      {/* Active plan summary */}
      <div className={styles.sectionLabel}>Current Plan</div>
      {activeBilling ? (
        <div className={styles.planSummary}>
          <span className={styles.planName}>{activeBilling.plan_name}</span>
          <Badge bg="success" className={styles.badge}>Active</Badge>
        </div>
      ) : (
        <p className={styles.emptyText}>No active plan assigned.</p>
      )}

      {/* Payment records */}
      <div className={styles.sectionLabel} style={{ marginTop: 20 }}>Payment History</div>
      {records.length === 0 ? (
        <p className={styles.emptyText}>No payment records yet.</p>
      ) : (
        <div className={styles.recordList}>
          {records.map(r => (
            <div key={r.id} className={styles.recordRow}>
              <div className={styles.recordLeft}>
                <span className={styles.recordDate}>{formatDate(r.due_date)}</span>
                <span className={styles.recordPlan}>{r.plan_name}</span>
              </div>
              <div className={styles.recordRight}>
                <span className={styles.recordAmount}>{formatPrice(r.amount)}</span>
                <Badge bg={STATUS_COLORS[r.status]} className={styles.badge}>
                  {r.status}
                </Badge>
                {r.status !== 'paid' && (
                  <Button
                    size="sm"
                    variant="outline-success"
                    className={styles.inlineBtn}
                    disabled={updatePayment.isPending}
                    onClick={() => updatePayment.mutate({ recordId: r.id!, status: 'paid' })}
                  >
                    Paid
                  </Button>
                )}
                {r.status !== 'overdue' && (
                  <Button
                    size="sm"
                    variant="outline-danger"
                    className={styles.inlineBtn}
                    disabled={updatePayment.isPending}
                    onClick={() => updatePayment.mutate({ recordId: r.id!, status: 'overdue' })}
                  >
                    Overdue
                  </Button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
