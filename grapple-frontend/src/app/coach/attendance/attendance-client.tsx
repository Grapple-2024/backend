'use client';

import { useState } from 'react';
import { Badge, Button, Form, Spinner } from 'react-bootstrap';
import { QRCodeSVG } from 'qrcode.react';
import { useGetCheckIns, useCheckIn, useDeleteCheckIn } from '@/hook/attendance';
import { useGetRequests } from '@/hook/request';
import { useGetGym } from '@/hook/gym';
import ConfirmationModal from '@/components/ConfirmationModal';
import styles from './Attendance.module.css';

function toISODate(d: Date) {
  return d.toISOString().slice(0, 10);
}

function formatTime(iso: string) {
  return new Date(iso).toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
}

export default function AttendanceClient() {
  const today = toISODate(new Date());
  const [date, setDate] = useState(today);
  const [selectedMemberId, setSelectedMemberId] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  const gym = useGetGym();
  const gymId = gym?.data?.id;
  const appUrl = process.env.NEXT_PUBLIC_APP_URL ?? '';
  const checkInUrl = `${appUrl}/checkin?gym_id=${gymId}`;

  const checkIns = useGetCheckIns({ date });
  const checkInMutation = useCheckIn();
  const deleteCheckIn = useDeleteCheckIn();

  // Get accepted members for the manual check-in dropdown
  const requests = useGetRequests([]);
  const acceptedMembers = (requests?.data?.data ?? []).filter((r: any) => r.status === 'Accepted');

  const handleManualCheckIn = () => {
    if (!selectedMemberId) return;
    const member = acceptedMembers.find((m: any) => m.requestor_id === selectedMemberId);
    if (!member) return;

    checkInMutation.mutate({
      member_id: member.requestor_id,
      member_name: `${member.first_name} ${member.last_name}`,
      avatar_url: member.profile?.avatar_url,
      method: 'manual',
    }, {
      onSuccess: () => setSelectedMemberId(''),
    });
  };

  const records = checkIns.data ?? [];

  return (
    <div className={styles.container}>
      <div className={styles.layout}>

        {/* Left: daily attendance list */}
        <div className={styles.main}>
          <div className={styles.header}>
            <div>
              <h4 className={styles.title}>Attendance</h4>
              <p className={styles.subtitle}>
                {records.length} check-in{records.length !== 1 ? 's' : ''} on {new Date(date + 'T12:00:00').toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })}
              </p>
            </div>
            <Form.Control
              type="date"
              value={date}
              onChange={e => setDate(e.target.value)}
              className={styles.datePicker}
            />
          </div>

          {/* Manual check-in bar */}
          <div className={styles.manualBar}>
            <Form.Select
              value={selectedMemberId}
              onChange={e => setSelectedMemberId(e.target.value)}
              className={styles.memberSelect}
            >
              <option value="">Select member to check in...</option>
              {acceptedMembers.map((m: any) => (
                <option key={m.requestor_id} value={m.requestor_id}>
                  {m.first_name} {m.last_name}
                </option>
              ))}
            </Form.Select>
            <Button
              variant="dark"
              onClick={handleManualCheckIn}
              disabled={!selectedMemberId || checkInMutation.isPending}
            >
              {checkInMutation.isPending ? <Spinner size="sm" /> : 'Check In'}
            </Button>
          </div>

          {/* Check-in list */}
          {checkIns.isPending ? (
            <div className={styles.loading}><Spinner animation="border" variant="dark" /></div>
          ) : records.length === 0 ? (
            <div className={styles.empty}>No check-ins for this date.</div>
          ) : (
            <div className={styles.list}>
              {records.map(r => (
                <div key={r.id} className={styles.checkInRow}>
                  <div className={styles.checkInLeft}>
                    {r.avatar_url ? (
                      <img src={r.avatar_url} alt={r.member_name} className={styles.avatar} />
                    ) : (
                      <div className={styles.avatarPlaceholder}>
                        {r.member_name?.[0] ?? '?'}
                      </div>
                    )}
                    <div className={styles.checkInInfo}>
                      <span className={styles.memberName}>{r.member_name}</span>
                      <span className={styles.checkInTime}>{formatTime(r.checked_in_at)}</span>
                    </div>
                  </div>
                  <div className={styles.checkInRight}>
                    <Badge bg={r.method === 'qr' ? 'info' : 'secondary'} className={styles.methodBadge}>
                      {r.method === 'qr' ? 'QR' : 'Manual'}
                    </Badge>
                    <Button
                      size="sm"
                      variant="outline-danger"
                      onClick={() => setDeleteTarget(r.id!)}
                    >
                      Remove
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Right: QR code panel */}
        <div className={styles.qrPanel}>
          <h6 className={styles.qrTitle}>Member Check-In QR</h6>
          <p className={styles.qrSubtitle}>
            Members scan this code to check themselves in.
          </p>
          {gymId ? (
            <>
              <div className={styles.qrWrapper}>
                <QRCodeSVG
                  value={checkInUrl}
                  size={180}
                  bgColor="#ffffff"
                  fgColor="#000000"
                  level="M"
                />
              </div>
              <p className={styles.qrUrl}>{checkInUrl}</p>
            </>
          ) : (
            <div className={styles.qrPlaceholder}>
              <Spinner size="sm" />
            </div>
          )}
        </div>
      </div>

      <ConfirmationModal
        show={!!deleteTarget}
        setShow={(v: boolean) => { if (!v) setDeleteTarget(null); }}
        onConfirm={() => {
          if (deleteTarget) {
            deleteCheckIn.mutate(deleteTarget, { onSuccess: () => setDeleteTarget(null) });
          }
        }}
      >
        Remove this check-in record?
      </ConfirmationModal>
    </div>
  );
}
