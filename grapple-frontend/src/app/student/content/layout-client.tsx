'use client';

import GrappleIcon from '@/components/GrappleIcon';
import DefaultDashboard from '@/layouts/DefaultLayout';

const studentSidebarData = [
  { title: 'My Gym', route: `/my-gym`, line: false, active: true, Icon: <GrappleIcon src='/my-gym-dark.svg' variant='dark' /> },
  { title: 'Content', route: `/content`, line: false, Icon: <GrappleIcon src='/content-light.svg' variant='light' /> },
  { title: 'Settings', route: `/settings`, line: false, Icon: <GrappleIcon src='/settings-dark.svg' variant='dark' /> },
];

function StudentContentLayout({ 
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
      defaultNode='Content'
      gym={gym}
      isCoach={false}
    >
      {children}
    </DefaultDashboard>
  );
}

export default StudentContentLayout;
