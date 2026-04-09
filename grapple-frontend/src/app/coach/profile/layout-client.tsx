'use client';

import GrappleIcon from "@/components/GrappleIcon";
import DefaultDashboard from "@/layouts/DefaultLayout";

const coachSidebarData = [
  { title: 'Dashboard',      route: `/dashboard`,  line: false, Icon: <GrappleIcon src='/dashboard-dark.svg'    variant='dark'  /> },
  { title: 'My Gym',         route: `/my-gym`,     line: false, Icon: <GrappleIcon src='/my-gym-dark.svg'       variant='dark'  /> },
  { title: 'User Management',route: `/user`,        line: false, Icon: <GrappleIcon src='/users-dark.svg'        variant='dark'  /> },
  { title: 'Attendance',     route: `/attendance`,  line: false, Icon: <GrappleIcon src='/settings-dark.svg'     variant='dark'  /> },
  { title: 'Content',        route: `/content`,     line: false, Icon: <GrappleIcon src='/content-dark.svg'      variant='dark'  /> },
  { title: 'Plans',          route: `/plans`,       line: false, Icon: <GrappleIcon src='/settings-dark.svg'     variant='dark'  /> },
  { title: 'Billing',        route: `/billing`,     line: false, Icon: <GrappleIcon src='/settings-dark.svg'     variant='dark'  /> },
  { title: 'Gym Profile',    route: `/profile`,     line: false, active: true, Icon: <GrappleIcon src='/gym-profile-light.svg' variant='light' /> },
  { title: 'Settings',       route: `/settings`,    line: false, Icon: <GrappleIcon src='/settings-dark.svg'     variant='dark'  /> },
];

function CoachProfileLayout({
  children,
  role,
  profile,
  nofitications,
  gym,
}: any) {
  return (
    <DefaultDashboard
      sidebar={coachSidebarData}
      defaultNode='Gym Profile'
      isCoach={true}
      gym={gym}
      profile={profile}
      notifications={nofitications}
    >
      {children}
    </DefaultDashboard>
  );
}

export default CoachProfileLayout;
