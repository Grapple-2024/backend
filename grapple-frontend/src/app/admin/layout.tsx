import { auth } from '@clerk/nextjs/server';
import { notFound, redirect } from 'next/navigation';

export const dynamic = 'force-dynamic';

const ADMIN_USER_ID = 'user_3BdfDoTV1og0ttLXHCLLNjk0EXJ';

export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  const { userId } = await auth();

  if (!userId) redirect('/auth');
  if (userId !== ADMIN_USER_ID) notFound();

  return <>{children}</>;
}
