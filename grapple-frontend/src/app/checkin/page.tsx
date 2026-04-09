'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { useAuth } from '@clerk/nextjs';
import { useRouter } from 'next/navigation';
import { Spinner } from 'react-bootstrap';
import { checkIn } from '@/api-requests/attendance';
import { useToken } from '@/hook/user';
import styles from './CheckIn.module.css';

type State = 'loading' | 'success' | 'already' | 'error';

export default function CheckInPage() {
  const searchParams = useSearchParams();
  const gymId = searchParams.get('gym_id');
  const { userId, isLoaded, isSignedIn } = useAuth();
  const router = useRouter();
  const token = useToken();
  const [state, setState] = useState<State>('loading');
  const [errorMsg, setErrorMsg] = useState('');

  // Redirect to sign-in if not authenticated
  useEffect(() => {
    if (!isLoaded) return;
    if (!isSignedIn) {
      const redirect = encodeURIComponent(`/checkin?gym_id=${gymId}`);
      router.replace(`/auth?redirect_url=${redirect}`);
    }
  }, [isLoaded, isSignedIn, gymId, router]);

  // Fire check-in once auth + token are ready
  useEffect(() => {
    if (!isSignedIn || !userId || !token || !gymId) return;

    checkIn({
      gym_id: gymId,
      member_id: userId,
      member_name: '',   // server resolves from profile if needed; blank is fine
      method: 'qr',
    }, token)
      .then(() => setState('success'))
      .catch((err: any) => {
        const msg = err?.response?.data?.error?.[0] ?? '';
        if (msg.includes('already checked in')) {
          setState('already');
        } else {
          setErrorMsg(msg || 'Something went wrong. Please try again.');
          setState('error');
        }
      });
  }, [isSignedIn, userId, token, gymId]);

  if (!gymId) {
    return (
      <div className={styles.container}>
        <div className={styles.card}>
          <div className={styles.icon}>❌</div>
          <h2 className={styles.heading}>Invalid Link</h2>
          <p className={styles.body}>This QR code is missing gym information. Ask your coach for a new one.</p>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.card}>
        {state === 'loading' && (
          <>
            <Spinner animation="border" className={styles.spinner} />
            <p className={styles.body}>Checking you in…</p>
          </>
        )}
        {state === 'success' && (
          <>
            <div className={styles.icon}>✅</div>
            <h2 className={styles.heading}>You're checked in!</h2>
            <p className={styles.body}>Your attendance has been recorded. See you on the mats.</p>
          </>
        )}
        {state === 'already' && (
          <>
            <div className={styles.icon}>👍</div>
            <h2 className={styles.heading}>Already checked in</h2>
            <p className={styles.body}>You've already been checked in today. See you on the mats.</p>
          </>
        )}
        {state === 'error' && (
          <>
            <div className={styles.icon}>❌</div>
            <h2 className={styles.heading}>Check-in failed</h2>
            <p className={styles.body}>{errorMsg}</p>
          </>
        )}
      </div>
    </div>
  );
}
