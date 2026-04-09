'use client';

import GrappleIcon from '@/components/GrappleIcon';
import DefaultDashboard from '@/layouts/DefaultLayout';

const studentSidebarData = [
  { title: 'My Gym', route: `/my-gym`, line: false, active: true, Icon: <GrappleIcon src='/my-gym-light.svg' variant='light' /> },
  { title: 'Content', route: `/content`, line: false, Icon: <GrappleIcon src='/content-dark.svg' variant='dark' /> },
  { title: 'Settings', route: `/settings`, line: false, Icon: <GrappleIcon src='/settings-dark.svg' variant='dark' /> },
];

function StudentMyGymLayout({ 
  gym, 
  children, 
  role, 
  profile,
  notifications,
}: any) {
  return (
    <DefaultDashboard 
      profile={profile}
      notifications={notifications}
      sidebar={studentSidebarData}
      defaultNode='My Gym'
      isCoach={false}
      gym={gym}
    >
      {children}
    </DefaultDashboard>
  );
}

export default StudentMyGymLayout;
