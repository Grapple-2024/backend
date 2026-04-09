// app/coach/layouts/DashboardClient.tsx
'use client';

import GrappleIcon from '@/components/GrappleIcon';
import DefaultDashboard from '@/layouts/DefaultLayout';

const coachSidebarData = [
  { title: 'Dashboard',      route: `/dashboard`,  line: false, Icon: <GrappleIcon src='/dashboard-dark.svg'    variant='dark'  /> },
  { title: 'My Gym',         route: `/my-gym`,     line: false, Icon: <GrappleIcon src='/my-gym-dark.svg'       variant='dark'  /> },
  { title: 'User Management',route: `/user`,        line: false, active: true, Icon: <GrappleIcon src='/users-light.svg' variant='light' /> },
  { title: 'Attendance',     route: `/attendance`,  line: false, Icon: <GrappleIcon src='/settings-dark.svg'     variant='dark'  /> },
  { title: 'Content',        route: `/content`,     line: false, Icon: <GrappleIcon src='/content-dark.svg'      variant='dark'  /> },
  { title: 'Plans',          route: `/plans`,       line: false, Icon: <GrappleIcon src='/settings-dark.svg'     variant='dark'  /> },
  { title: 'Billing',        route: `/billing`,     line: false, Icon: <GrappleIcon src='/settings-dark.svg'     variant='dark'  /> },
  { title: 'Gym Profile',    route: `/profile`,     line: false, Icon: <GrappleIcon src='/gym-profile-dark.svg'  variant='dark'  /> },
  { title: 'Settings',       route: `/settings`,    line: false, Icon: <GrappleIcon src='/settings-dark.svg'     variant='dark'  /> },
];

interface DashboardClientProps {
  gym: any;
  children: React.ReactNode;
  role: string | null;
  profile: any;
  notifications: any;
}

function UserDashboardClient({
  gym,
  children,
  role,
  profile,
  notifications
}: DashboardClientProps) {
  return (
    <DefaultDashboard
      sidebar={coachSidebarData}
      profile={profile}
      defaultNode='User Management'
      isCoach={true}
      gym={gym}
      notifications={notifications}
    >
      {children}
    </DefaultDashboard>
  );
}

export default UserDashboardClient;
