'use client';

import { useRouter } from 'next/navigation';
import DashboardLayout from '../DashboardLayout';
import { useGetUserProfile } from '@/hook/profile';


function Dashboard({
  children,
  sidebar,
  defaultNode,
  gym,
  isCoach
}: Readonly<{
  children: React.ReactNode;
  sidebar?: { title: string; route: string; line: boolean; active?: boolean; }[];
  defaultNode: string;
  gym: any;
  isCoach: boolean;
}>) {
  const router = useRouter();

  const { isPending } = useGetUserProfile();

  if (isPending) {
    return null;
  }

  return (
    <>
      <DashboardLayout
        defaultNode={defaultNode}
        sidebarData={sidebar}
        gymName={gym?.data?.name as string}
        isStudent={!isCoach}
        profile={null}
        gym={gym}
        notifications={null}
      >
        {children}
      </DashboardLayout>
    </>
  );
}

export default Dashboard;
