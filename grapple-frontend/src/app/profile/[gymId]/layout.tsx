'use client';

import dynamic from 'next/dynamic';
import styles from './styles.module.css';
import { useMobileContext } from '@/context/mobile';
import { Container } from 'react-bootstrap';
import { useSignOut } from '@/hook/auth';
import { useGetUserProfile } from '@/hook/profile';
import { useIsSignedIn } from '@/hook/user';

const Navigation = dynamic(() => import('../../../components/Navigation'), { ssr: false });
const MobileNavigation = dynamic(() => import('../../../components/MobileNavigation'), { ssr: false });



function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const { isMobile } = useMobileContext();
  const profile = useGetUserProfile();
  const isSignedIn = useIsSignedIn();
  
  const mutation = useSignOut();
  
  if (!isSignedIn) {
    return (
      <>
        <div className={styles.mainContainer}>
          <div className={styles.navigationContainer}>
            {
              isMobile ? (
                <>
                  <MobileNavigation
                    avatarUrl={profile?.data?.avatar_url ? profile?.data?.avatar_url: ""}
                    isSignedIn={!!isSignedIn}
                    onSignOut={() => {
                      mutation.mutate();
                    }} 
                  />
                </>
              ) : (
                <>
                  <Navigation 
                    onSignOut={() => {
                      mutation.mutate();
                    }} 
                    hasArrow={false}
                    avatarUrl={profile?.data?.avatar_url ? profile?.data?.avatar_url : ""}
                  />
                </>
              )
            }
          </div>
          <div style={{ margin: 0, backgroundColor: '#F1F5F9' }}>
            {children}
          </div>
        </div>
      </>
    );
  }
  
  return (
    <>
      <div className={styles.mainContainer}>
        <div className={styles.navigationContainer}>
          {
            isMobile ? (
              <>
                <MobileNavigation
                  avatarUrl={profile?.data?.avatar_url ? profile?.data?.avatar_url: ""}
                  isSignedIn={!!profile?.data}
                />
              </>
            ) : (
              <>
                <Navigation 
                  hasArrow={false}
                  avatarUrl={profile?.data?.avatar_url ? profile?.data?.avatar_url : ""}
                />
              </>
            )
          }
        </div>
        <div style={{ margin: 0, backgroundColor: '#F1F5F9' }}>
          {children}
        </div>
      </div>
    </>
  );
}

export default RootLayout;
