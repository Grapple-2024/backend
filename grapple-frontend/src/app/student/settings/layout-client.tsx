'use client';

import GrappleIcon from '@/components/GrappleIcon';
import DefaultDashboard from '@/layouts/DefaultLayout';

const studentSidebarData = [
  { title: 'My Gym', route: `/my-gym`, line: false, active: true, Icon: <GrappleIcon src='/my-gym-dark.svg' variant='dark' /> },
  { title: 'Content', route: `/content`, line: false, Icon: <GrappleIcon src='/content-dark.svg' variant='dark' /> },
  { title: 'Settings', route: `/settings`, line: false, Icon: <GrappleIcon src='/settings-light.svg' variant='light' /> },
];

function StudentSettingsLayout({ 
  gym, 
  children, 
  role, 
  profile,
  notifications
}: any) {
  return (
    <DefaultDashboard 
      profile={profile}
      notifications={notifications}
      sidebar={studentSidebarData}
      defaultNode='Settings'
      isCoach={false}
      gym={gym}
    >
      {children}
    </DefaultDashboard>
  );
}

export default StudentSettingsLayout;
