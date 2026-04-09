import { auth } from '@clerk/nextjs/server';
import { redirect } from 'next/navigation';
import PlansClient from './plans-client';

export const dynamic = 'force-dynamic';

export default async function PlansPage() {
  const { userId } = await auth();
  if (!userId) redirect('/auth');

  return <PlansClient />;
}
