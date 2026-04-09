import { auth } from '@clerk/nextjs/server';
import { redirect } from 'next/navigation';
import StudentMyGymPage from './page-client';

export const dynamic = 'force-dynamic';

export default async function UsersPage() {
  const { userId } = await auth();
  if (!userId) redirect('/auth');

  return <StudentMyGymPage announcements={[]} isAcceptedUser={false} />;
}
