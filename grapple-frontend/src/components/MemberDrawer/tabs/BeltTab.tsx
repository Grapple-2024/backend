'use client';

import { useState } from 'react';
import { Badge, Button, Spinner } from 'react-bootstrap';
import { User } from '@/components/UserTable';
import { useGetPromotionHistory, useDeletePromotion } from '@/hook/promotion';
import BeltBadge from '@/components/BeltBadge';
import PromotionModal from '@/components/PromotionModal';
import styles from '../MemberDrawer.module.css';

interface Props {
  member: User;
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-US', {
    month: 'short', day: 'numeric', year: 'numeric',
  });
}

export default function BeltTab({ member }: Props) {
  const [showModal, setShowModal] = useState(false);
  const fullName = `${member.first_name} ${member.last_name}`;
  const history = useGetPromotionHistory(member.requestor_id);
  const deletePromotion = useDeletePromotion(member.requestor_id);

  if (history.isPending) {
    return <div className={styles.tabLoading}><Spinner size="sm" /></div>;
  }

  const records = history.data ?? [];
  const current = records[0];

  return (
    <div className={styles.tabContent}>
      {/* Current rank */}
      <div className={styles.sectionLabel}>Current Rank</div>
      {current ? (
        <div style={{ marginBottom: 24, display: 'flex', alignItems: 'center', gap: 10 }}>
          <BeltBadge system={current.system as 'adult' | 'kids'} belt={current.belt} stripes={current.stripes} />
        </div>
      ) : (
        <p className={styles.emptyText} style={{ marginBottom: 24 }}>No promotions recorded yet.</p>
      )}

      {/* Record promotion button */}
      <Button
        variant="dark"
        size="sm"
        style={{ marginBottom: 24 }}
        onClick={() => setShowModal(true)}
      >
        + Record Promotion
      </Button>

      {/* Promotion history */}
      <div className={styles.sectionLabel}>Promotion History ({records.length})</div>
      {records.length === 0 ? (
        <p className={styles.emptyText}>No promotions recorded.</p>
      ) : (
        <div className={styles.recordList}>
          {records.map(r => (
            <div key={r.id} className={styles.recordRow}>
              <div className={styles.recordLeft}>
                <BeltBadge
                  system={r.system as 'adult' | 'kids'}
                  belt={r.belt}
                  stripes={r.stripes}
                />
                <span className={styles.recordPlan}>{formatDate(r.promoted_at)}{r.promoted_by ? ` · ${r.promoted_by}` : ''}</span>
                {r.notes && (
                  <span className={styles.recordPlan} style={{ fontStyle: 'italic' }}>{r.notes}</span>
                )}
              </div>
              <div className={styles.recordRight}>
                <Badge bg={r.system === 'adult' ? 'dark' : 'secondary'} className={styles.badge}>
                  {r.system}
                </Badge>
                <Button
                  size="sm"
                  variant="outline-danger"
                  className={styles.inlineBtn}
                  disabled={deletePromotion.isPending}
                  onClick={() => deletePromotion.mutate(r.id!)}
                >
                  Remove
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <PromotionModal
        show={showModal}
        onHide={() => setShowModal(false)}
        memberId={member.requestor_id}
        memberName={fullName}
        avatarUrl={member.profile?.avatar_url}
      />
    </div>
  );
}
