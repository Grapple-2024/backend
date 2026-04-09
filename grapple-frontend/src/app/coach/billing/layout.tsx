import { auth } from '@clerk/nextjs/server';
import { redirect } from 'next/navigation';
import BillingLayout from './layout-client';
import axios from 'axios';

export const dynamic = 'force-dynamic';

export default async function CoachBillingLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const { userId, getToken } = await auth();
  if (!userId) redirect('/auth');

  const token = await getToken();

  let profile = null;
  try {
    const { data } = await axios.get(`${process.env.NEXT_PUBLIC_API_HOST}/profiles`, {
      params: { current_user: true },
      headers: { Authorization: `Bearer ${token}` },
    });
    profile = data;
  } catch {
    // no profile yet
  }

  return (
    <BillingLayout gym={null} role={null} profile={profile} notifications={[]}>
      {children}
    </BillingLayout>
  );
}
