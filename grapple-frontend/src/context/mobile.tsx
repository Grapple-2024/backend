import { useState, useContext, createContext, useEffect } from 'react';

const initialState: any = {
  isMobile: false,
  setIsMobile: (val: boolean) => {},
};

// Step 1: Create a new context
const MobileContext = createContext(initialState);

// Step 2: Define a provider component
export const MobileProvider = ({ children }: any) => {
  const [isMobile, setIsMobile] = useState(false);

  const handleResize = () => {
    if (typeof window !== 'undefined') {
      const isLandscape = window.matchMedia('(orientation: landscape)').matches;
      const isActualLandscape = isLandscape && window.innerHeight <= 768 && window.innerWidth <= 1000;

      setIsMobile(window.innerWidth <= 768 || isActualLandscape);
    }
  };

  useEffect(() => {
    // Initial check
    handleResize();

    // Listen for resize events
    if (typeof window !== 'undefined') {
      window.addEventListener('resize', handleResize);
    }

    // Cleanup
    return () => {
      if (typeof window !== 'undefined') {
        window.removeEventListener('resize', handleResize);
      }
    };
  }, []);

  return (
    <MobileContext.Provider value={{ isMobile, setIsMobile }}>
      {children}
    </MobileContext.Provider>
  );
};

// Step 3: Create a custom hook to use this context
export const useMobileContext = () => useContext(MobileContext);