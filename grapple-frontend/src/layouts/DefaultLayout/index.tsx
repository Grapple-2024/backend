'use client';

import DashboardLayout from '../DashboardLayout';


function DefaultDashboard({
  children,
  sidebar,
  defaultNode,
  gym,
  isCoach,
  profile = null,
  notifications = null,
}: Readonly<{
  children: React.ReactNode;
  sidebar?: { title: string; route: string; line: boolean; active?: boolean; }[];
  defaultNode: string;
  gym: any;
  isCoach: boolean;
  profile: any;
  notifications: any;
}>) {
  return (
    <>
      <DashboardLayout
        gym={gym}
        profile={profile}
        defaultNode={defaultNode}
        sidebarData={sidebar}
        gymName={gym?.data?.name as string}
        isStudent={!isCoach}
        notifications={notifications}
      >
        {children}
      </DashboardLayout>
    </>
  );
}

export default DefaultDashboard;
