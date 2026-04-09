'use client';

import styles from './EmptyLayout.module.css';

import { Container } from 'react-bootstrap';
import dynamic from 'next/dynamic';
import { useSignOut } from '@/hook/auth';
import { useGetUserProfile } from '@/hook/profile';

const Navigation = dynamic(() => import('../../components/Navigation'), { ssr: false });

function EmptyLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const mutation = useSignOut();
  const { error, data } = useGetUserProfile();

  const firstName = data?.first_name;
  const lastName = data?.last_name;

  return (
    <>
      <Navigation 
        onSignOut={() => {
          mutation.mutate();
        }} 
        isSignedIn={!error}
        hasArrow={false}
        userName={`${firstName} ${lastName}`}
      />
      <Container className={styles.container}>
        {children}
      </Container>
    </>
  );
}

export default EmptyLayout;
