import { Badge } from 'react-bootstrap';
import { User } from '@/components/UserTable';
import styles from '../MemberDrawer.module.css';

interface Props {
  member: User;
}

export default function OverviewTab({ member }: Props) {
  const fullName = `${member.first_name} ${member.last_name}`;

  return (
    <div className={styles.tabContent}>
      <div className={styles.overviewRow}>
        <span className={styles.overviewLabel}>Name</span>
        <span className={styles.overviewValue}>{fullName}</span>
      </div>
      <div className={styles.overviewRow}>
        <span className={styles.overviewLabel}>Email</span>
        <span className={styles.overviewValue}>{member.requestor_email || '—'}</span>
      </div>
      <div className={styles.overviewRow}>
        <span className={styles.overviewLabel}>Phone</span>
        <span className={styles.overviewValue}>{member.profile?.phone_number || '—'}</span>
      </div>
      <div className={styles.overviewRow}>
        <span className={styles.overviewLabel}>Membership</span>
        <span className={styles.overviewValue}>
          {member.membership_type ? (
            <Badge bg="secondary" className={styles.badge}>
              {member.membership_type.toLowerCase()}
            </Badge>
          ) : '—'}
        </span>
      </div>
      <div className={styles.overviewRow}>
        <span className={styles.overviewLabel}>Status</span>
        <span className={styles.overviewValue}>
          <Badge bg={member.status === 'Accepted' ? 'success' : 'warning'} className={styles.badge}>
            {member.status || '—'}
          </Badge>
        </span>
      </div>
      <div className={styles.overviewRow}>
        <span className={styles.overviewLabel}>Role</span>
        <span className={styles.overviewValue}>{member.role || '—'}</span>
      </div>
    </div>
  );
}
