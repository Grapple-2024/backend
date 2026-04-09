import { createContext, useState } from "react";

export const SelectionContext = createContext({
  selection: '',
  setSelection: (value: string) => {},
});

export const SelectionProvider = ({ children }: any) => {
  const [selection, setSelection] = useState('Search for a gym');

  return (
    <SelectionContext.Provider value={{ 
      selection, 
      setSelection,
    }}>
      {children}
    </SelectionContext.Provider>
  );
};