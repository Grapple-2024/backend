import { auth } from '@clerk/nextjs/server';
import { redirect } from 'next/navigation';
import UsersPageClient from './client-page';

export const dynamic = 'force-dynamic';

export default async function UsersPage() {
  const { userId } = await auth();
  if (!userId) redirect('/auth');

  return <UsersPageClient requests={[]} role={null} gym={null} />;
}
