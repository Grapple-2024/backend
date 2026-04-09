import { Badge, Button, Spinner } from 'react-bootstrap';
import { User } from '@/components/UserTable';
import { useGetCheckIns, useDeleteCheckIn } from '@/hook/attendance';
import styles from '../MemberDrawer.module.css';

interface Props {
  member: User;
}

function formatDateTime(iso: string) {
  return new Date(iso).toLocaleString('en-US', {
    month: 'short', day: 'numeric', year: 'numeric',
    hour: 'numeric', minute: '2-digit',
  });
}

export default function AttendanceTab({ member }: Props) {
  const checkIns = useGetCheckIns({ member_id: member.requestor_id });
  const deleteCheckIn = useDeleteCheckIn();

  if (checkIns.isPending) {
    return <div className={styles.tabLoading}><Spinner size="sm" /></div>;
  }

  const records = checkIns.data ?? [];

  return (
    <div className={styles.tabContent}>
      <div className={styles.sectionLabel}>
        Last 30 days — {records.length} check-in{records.length !== 1 ? 's' : ''}
      </div>
      {records.length === 0 ? (
        <p className={styles.emptyText}>No check-ins recorded.</p>
      ) : (
        <div className={styles.recordList}>
          {records.map(r => (
            <div key={r.id} className={styles.recordRow}>
              <div className={styles.recordLeft}>
                <span className={styles.recordDate}>{formatDateTime(r.checked_in_at)}</span>
              </div>
              <div className={styles.recordRight}>
                <Badge
                  bg={r.method === 'qr' ? 'info' : 'secondary'}
                  className={styles.badge}
                >
                  {r.method === 'qr' ? 'QR' : 'Manual'}
                </Badge>
                <Button
                  size="sm"
                  variant="outline-danger"
                  className={styles.inlineBtn}
                  disabled={deleteCheckIn.isPending}
                  onClick={() => deleteCheckIn.mutate(r.id!)}
                >
                  Remove
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
