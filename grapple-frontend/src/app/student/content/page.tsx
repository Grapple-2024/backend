import { auth } from '@clerk/nextjs/server';
import { redirect } from 'next/navigation';
import StudentContentPage from './page-client';

export const dynamic = 'force-dynamic';

export default async function ContentPage() {
  const { userId } = await auth();
  if (!userId) redirect('/auth');

  return <StudentContentPage series={[]} />;
}
