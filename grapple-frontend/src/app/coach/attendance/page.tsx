import { auth } from '@clerk/nextjs/server';
import { redirect } from 'next/navigation';
import AttendanceClient from './attendance-client';

export const dynamic = 'force-dynamic';

export default async function AttendancePage() {
  const { userId } = await auth();
  if (!userId) redirect('/auth');

  return <AttendanceClient />;
}
