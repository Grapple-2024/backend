import { useState, useContext, createContext, useEffect } from 'react';

const initialState: any = {
  loading: false, 
  setLoading: (val: boolean) => {},
};

// Step 1: Create a new context
const LoadingContext = createContext(initialState);

// Step 2: Define a provider component
export const LoadingProvider = ({ children }: any) => {
  const [loading, setLoading] = useState(false);

  return (
    <LoadingContext.Provider value={{ loading, setLoading }}>
      {children}
    </LoadingContext.Provider>
  );
};

// Step 3: Create a custom hook to use this context
export const useLoadingContext = () => useContext(LoadingContext);