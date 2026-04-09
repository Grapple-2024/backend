import { auth } from '@clerk/nextjs/server';
import { redirect } from 'next/navigation';
import StudentContentLayout from './layout-client';
import axios from 'axios';

export const dynamic = 'force-dynamic';

export default async function CoachUserLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const { userId, getToken } = await auth();
  if (!userId) redirect('/auth');

  const token = await getToken();

  const { data: profile } = await axios.get(`${process.env.NEXT_PUBLIC_API_HOST}/profiles`, {
    params: { current_user: true },
    headers: { Authorization: `Bearer ${token}` }
  });

  return (
    <StudentContentLayout
      gym={null}
      role={null}
      profile={profile}
      notifications={[]}
    >
      {children}
    </StudentContentLayout>
  );
}
