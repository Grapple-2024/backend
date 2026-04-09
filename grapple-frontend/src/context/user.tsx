import { useState, useContext, createContext } from 'react';

const initialState: any = {
  profile: null,
  setProfile: (val: any) => {},
  role: null,
  setRole: (val: any) => {},
  currentGym: null,
  setCurrentGym: (val: any) => {},
  gymsArray: [],
  setGymsArray: (val: any) => {},
};

// Step 1: Create a new context
const UserContext = createContext(initialState);

// Step 2: Define a provider component
export const UserProvider = ({ children }: any) => {
  const [profile, setProfile] = useState(null);
  const [role, setRole] = useState(null);
  const [currentGym, setCurrentGym] = useState(null);
  const [gymsArray, setGymsArray] = useState([]);

  return (
    <UserContext.Provider value={{ 
      profile, 
      setProfile,
      role,
      setRole,
      currentGym,
      setCurrentGym,
      gymsArray,
      setGymsArray,
    }}>
      {children}
    </UserContext.Provider>
  );
};

// Step 3: Create a custom hook to use this context
export const useUserContext = () => useContext(UserContext);