import { auth } from '@clerk/nextjs/server';
import { redirect } from 'next/navigation';
import ClientContentPage from './client-page';

export const dynamic = 'force-dynamic';

export default async function ContentPage() {
  const { userId } = await auth();
  if (!userId) redirect('/auth');

  return <ClientContentPage series={[]} />;
}
