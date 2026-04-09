import { auth } from '@clerk/nextjs/server';
import { redirect } from 'next/navigation';
import StudentMyGymLayout from './layout-client';

export const dynamic = 'force-dynamic';

export default async function StudentUserLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const { userId } = await auth();
  if (!userId) redirect('/auth');

  return (
    <StudentMyGymLayout
      gym={null}
      role={null}
      profile={null}
      notifications={[]}
    >
      {children}
    </StudentMyGymLayout>
  );
}
