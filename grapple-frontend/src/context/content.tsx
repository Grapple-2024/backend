import VideoDashboardCreateModal from '@/components/VideoDashboardCreateModal';
import { useState, useContext, createContext, useEffect } from 'react';

interface ContentState {
  open: boolean;
  setOpen: (val: boolean) => void;
}

const initialState: ContentState = {
  open: false,
  setOpen: (val: boolean) => {},
};

const ContentContext = createContext(initialState);

export const ContentProvider = ({ children }: any) => {
  const [open, setOpen] = useState(false);

  return (
    <ContentContext.Provider value={{ 
      open, 
      setOpen,
    }}>
      {children}
    </ContentContext.Provider>
  );
};

export const useContentContext = () => useContext(ContentContext);