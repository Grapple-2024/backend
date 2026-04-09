import Warning from '@/components/Warning';
import { useState, useContext, createContext, useEffect } from 'react';

const initialState: any = {
  show: false,
  color: '',
  message: '',
  setShow: (val: boolean) => {},
  setColor: (val: string) => {},
  setMessage: (val: string) => {},
};

// Step 1: Create a new context
const MessagingContext = createContext(initialState);

// Step 2: Define a provider component
export const MessagingProvider = ({ children }: any) => {
  const [show, setShow] = useState(false);
  const [color, setColor] = useState('danger');
  const [message, setMessage] = useState('');

  useEffect(() => {
    setTimeout(() => {
      setShow(false);
    }, 6000);
  }, [message, show, color]);

  return (
    <MessagingContext.Provider value={{ show, setShow, color, setColor, message, setMessage }}>
      {children}
      <Warning message={message} show={show} bg={color} />
    </MessagingContext.Provider>
  );
};

// Step 3: Create a custom hook to use this context
export const useMessagingContext = () => useContext(MessagingContext);