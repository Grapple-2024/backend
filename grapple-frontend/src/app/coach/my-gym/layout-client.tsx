'use client';

import GrappleIcon from '@/components/GrappleIcon';
import DefaultDashboard from '@/layouts/DefaultLayout';

const coachSidebarData = [
  { title: 'Dashboard',      route: `/dashboard`,  line: false, Icon: <GrappleIcon src='/dashboard-dark.svg'    variant='dark'  /> },
  { title: 'My Gym',         route: `/my-gym`,     line: false, active: true, Icon: <GrappleIcon src='/my-gym-light.svg' variant='light' /> },
  { title: 'User Management',route: `/user`,        line: false, Icon: <GrappleIcon src='/users-dark.svg'        variant='dark'  /> },
  { title: 'Attendance',     route: `/attendance`,  line: false, Icon: <GrappleIcon src='/settings-dark.svg'     variant='dark'  /> },
  { title: 'Content',        route: `/content`,     line: false, Icon: <GrappleIcon src='/content-dark.svg'      variant='dark'  /> },
  { title: 'Plans',          route: `/plans`,       line: false, Icon: <GrappleIcon src='/settings-dark.svg'     variant='dark'  /> },
  { title: 'Billing',        route: `/billing`,     line: false, Icon: <GrappleIcon src='/settings-dark.svg'     variant='dark'  /> },
  { title: 'Gym Profile',    route: `/profile`,     line: false, Icon: <GrappleIcon src='/gym-profile-dark.svg'  variant='dark'  /> },
  { title: 'Settings',       route: `/settings`,    line: false, Icon: <GrappleIcon src='/settings-dark.svg'     variant='dark'  /> },
];

function CoachMyGymLayout({
  children,
  role,
  profile,
  nofitications,
  gym,
}: any) {
  return (
    <DefaultDashboard
      sidebar={coachSidebarData}
      defaultNode='My Gym'
      isCoach={true}
      gym={gym}
      profile={profile}
      notifications={nofitications}
    >
      {children}
    </DefaultDashboard>
  );
}

export default CoachMyGymLayout;
