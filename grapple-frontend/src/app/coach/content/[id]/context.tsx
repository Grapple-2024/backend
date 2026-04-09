import { useState, useContext, createContext } from 'react';

const initialState: any = {
  currentVideoUrl: '',
  setCurrentVideoUrl: (val: string) => {},
};

// Step 1: Create a new context
const ContentPageContext = createContext(initialState);

// Step 2: Define a provider component
export const ContentPageProvider = ({ children }: any) => {
  const [currentVideoUrl, setCurrentVideoUrl] = useState('');

  return (
    <ContentPageContext.Provider value={{ currentVideoUrl, setCurrentVideoUrl }}>
      {children}
    </ContentPageContext.Provider>
  );
};

// Step 3: Create a custom hook to use this context
export const useContentPageProvider = () => useContext(ContentPageContext);