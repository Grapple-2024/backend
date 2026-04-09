import { auth } from '@clerk/nextjs/server';
import { redirect } from 'next/navigation';
import BillingClient from './billing-client';

export const dynamic = 'force-dynamic';

export default async function BillingPage() {
  const { userId } = await auth();
  if (!userId) redirect('/auth');

  return <BillingClient />;
}
