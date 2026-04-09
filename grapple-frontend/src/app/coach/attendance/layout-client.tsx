'use client';

import GrappleIcon from '@/components/GrappleIcon';
import DefaultDashboard from '@/layouts/DefaultLayout';

const coachSidebarData = [
  { title: 'Dashboard',      route: `/dashboard`,   line: false, Icon: <GrappleIcon src='/dashboard-dark.svg'    variant='dark'  /> },
  { title: 'My Gym',         route: `/my-gym`,      line: false, Icon: <GrappleIcon src='/my-gym-dark.svg'       variant='dark'  /> },
  { title: 'User Management',route: `/user`,         line: false, Icon: <GrappleIcon src='/users-dark.svg'        variant='dark'  /> },
  { title: 'Attendance',     route: `/attendance`,   line: false, active: true, Icon: <GrappleIcon src='/settings-light.svg' variant='light' /> },
  { title: 'Content',        route: `/content`,     line: false, Icon: <GrappleIcon src='/content-dark.svg'       variant='dark'  /> },
  { title: 'Plans',          route: `/plans`,       line: false, Icon: <GrappleIcon src='/settings-dark.svg'      variant='dark'  /> },
  { title: 'Billing',        route: `/billing`,     line: false, Icon: <GrappleIcon src='/settings-dark.svg'      variant='dark'  /> },
  { title: 'Gym Profile',    route: `/profile`,     line: false, Icon: <GrappleIcon src='/gym-profile-dark.svg'   variant='dark'  /> },
  { title: 'Settings',       route: `/settings`,    line: false, Icon: <GrappleIcon src='/settings-dark.svg'      variant='dark'  /> },
];

function AttendanceLayout({ children, profile, notifications, gym }: any) {
  return (
    <DefaultDashboard
      sidebar={coachSidebarData}
      defaultNode='Attendance'
      isCoach={true}
      gym={gym}
      profile={profile}
      notifications={notifications}
    >
      {children}
    </DefaultDashboard>
  );
}

export default AttendanceLayout;
