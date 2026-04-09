import { useAuth } from '@clerk/nextjs';
import { useState, useEffect } from 'react';

export const useToken = (): string => {
  const { getToken } = useAuth();
  const [token, setToken] = useState('');

  useEffect(() => {
    getToken().then(t => setToken(t || ''));
  }, [getToken]);

  return token;
};

export const useIsSignedIn = (): boolean => {
  const { isSignedIn } = useAuth();
  return !!isSignedIn;
};
