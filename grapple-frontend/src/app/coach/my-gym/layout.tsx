import { auth } from '@clerk/nextjs/server';
import { redirect } from 'next/navigation';
import CoachMyGymLayout from './layout-client';
import { getRequests } from '@/api-requests/request';
import axios from 'axios';

export const dynamic = 'force-dynamic';

export default async function Layout({
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
      headers: { Authorization: `Bearer ${token}` }
    });
    profile = data;
  } catch {
    // New users or users with no backend profile yet — render without profile
  }

  return (
    <CoachMyGymLayout
      gym={null}
      role={null}
      profile={profile}
      notifications={[]}
    >
      {children}
    </CoachMyGymLayout>
  );
}
