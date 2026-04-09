'use client';

import { useState } from 'react';
import { Modal, Button, Form, Spinner } from 'react-bootstrap';
import { ADULT_BELTS, KIDS_BELTS, BeltSystem, Promotion } from '@/api-requests/promotion';
import { useRecordPromotion } from '@/hook/promotion';
import { useGetGym } from '@/hook/gym';
import BeltBadge from '@/components/BeltBadge';
import styles from './PromotionModal.module.css';

interface Props {
  show: boolean;
  onHide: () => void;
  memberId: string;
  memberName: string;
  avatarUrl?: string;
  promotedBy?: string;
}

export default function PromotionModal({
  show,
  onHide,
  memberId,
  memberName,
  avatarUrl,
  promotedBy = '',
}: Props) {
  const gym = useGetGym();
  const recordPromotion = useRecordPromotion();

  const [system, setSystem] = useState<BeltSystem>('adult');
  const [belt, setBelt] = useState('');
  const [stripes, setStripes] = useState(0);
  const [promotedAt, setPromotedAt] = useState(() => new Date().toISOString().slice(0, 10));
  const [notes, setNotes] = useState('');
  const [promotedByVal, setPromotedByVal] = useState(promotedBy);

  const belts = system === 'adult' ? ADULT_BELTS : KIDS_BELTS;

  const handleSystemChange = (s: BeltSystem) => {
    setSystem(s);
    setBelt('');
    setStripes(0);
  };

  const handleSubmit = () => {
    if (!belt || !gym.data?.id) return;
    const payload: Omit<Promotion, 'id' | 'created_at'> = {
      gym_id: gym.data.id,
      member_id: memberId,
      member_name: memberName,
      avatar_url: avatarUrl,
      system,
      belt,
      stripes,
      promoted_at: new Date(promotedAt).toISOString(),
      notes: notes.trim() || undefined,
      promoted_by: promotedByVal.trim() || undefined,
    };
    recordPromotion.mutate(payload, {
      onSuccess: () => {
        handleClose();
      },
    });
  };

  const handleClose = () => {
    setSystem('adult');
    setBelt('');
    setStripes(0);
    setPromotedAt(new Date().toISOString().slice(0, 10));
    setNotes('');
    setPromotedByVal(promotedBy);
    onHide();
  };

  return (
    <Modal show={show} onHide={handleClose} centered>
      <Modal.Header closeButton style={{ background: '#000', color: '#fff' }}>
        <Modal.Title style={{ fontSize: 16 }}>
          Record Promotion — {memberName}
        </Modal.Title>
      </Modal.Header>
      <Modal.Body style={{ padding: 24 }}>
        {/* System toggle */}
        <div className={styles.systemToggle}>
          <button
            className={`${styles.systemBtn} ${system === 'adult' ? styles.systemBtnActive : ''}`}
            onClick={() => handleSystemChange('adult')}
            type="button"
          >
            Adult BJJ
          </button>
          <button
            className={`${styles.systemBtn} ${system === 'kids' ? styles.systemBtnActive : ''}`}
            onClick={() => handleSystemChange('kids')}
            type="button"
          >
            Kids BJJ
          </button>
        </div>

        {/* Belt selection */}
        <div className={styles.formLabel}>Belt</div>
        <div className={styles.beltGrid}>
          {belts.map(b => (
            <button
              key={b}
              type="button"
              className={`${styles.beltOption} ${belt === b ? styles.beltOptionSelected : ''}`}
              onClick={() => setBelt(b)}
            >
              <BeltBadge system={system} belt={b} stripes={0} showLabel={false} />
              <span style={{ textTransform: 'capitalize', fontSize: 12 }}>{b}</span>
            </button>
          ))}
        </div>

        {/* Stripes */}
        <div className={styles.formLabel} style={{ marginTop: 20 }}>Stripes</div>
        <div className={styles.stripeRow}>
          {[0, 1, 2, 3, 4].map(n => (
            <button
              key={n}
              type="button"
              className={`${styles.stripeBtn} ${stripes === n ? styles.stripeBtnSelected : ''}`}
              onClick={() => setStripes(n)}
            >
              {n}
            </button>
          ))}
        </div>

        {/* Preview */}
        {belt && (
          <div style={{ marginTop: 16, padding: '10px 14px', background: '#f7f7f7', borderRadius: 8 }}>
            <BeltBadge system={system} belt={belt} stripes={stripes} />
          </div>
        )}

        {/* Date */}
        <Form.Group style={{ marginTop: 20 }}>
          <Form.Label className={styles.formLabel}>Date Promoted</Form.Label>
          <Form.Control
            type="date"
            value={promotedAt}
            onChange={e => setPromotedAt(e.target.value)}
          />
        </Form.Group>

        {/* Promoted by */}
        <Form.Group style={{ marginTop: 12 }}>
          <Form.Label className={styles.formLabel}>Promoted By</Form.Label>
          <Form.Control
            type="text"
            placeholder="Coach name"
            value={promotedByVal}
            onChange={e => setPromotedByVal(e.target.value)}
          />
        </Form.Group>

        {/* Notes */}
        <Form.Group style={{ marginTop: 12 }}>
          <Form.Label className={styles.formLabel}>Notes (optional)</Form.Label>
          <Form.Control
            as="textarea"
            rows={2}
            placeholder="Tournament, ceremony details, etc."
            value={notes}
            onChange={e => setNotes(e.target.value)}
          />
        </Form.Group>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="outline-secondary" onClick={handleClose}>
          Cancel
        </Button>
        <Button
          variant="dark"
          disabled={!belt || recordPromotion.isPending}
          onClick={handleSubmit}
        >
          {recordPromotion.isPending ? <Spinner size="sm" /> : 'Record Promotion'}
        </Button>
      </Modal.Footer>
    </Modal>
  );
}
