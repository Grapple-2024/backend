import { auth } from '@clerk/nextjs/server';
import { redirect } from 'next/navigation';
import MyGymPage from './page-client';

export const dynamic = 'force-dynamic';

export default async function UsersPage() {
  const { userId } = await auth();
  if (!userId) redirect('/auth');

  return <MyGymPage announcements={[]} />;
}
