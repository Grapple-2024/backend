import { useState, useContext, createContext } from 'react';

interface EditSeriesState {
  currentSeries: any;
  setCurrentSeries: (val: any) => void;
  isEditing: boolean;
  setIsEditing: (val: boolean) => void; 
  currentSelection: any;
  setCurrentSelection: (val: any) => void;
  step: number;
  setStep: (val: any) => void;
  series: any;
  setSeries: (val: any) => void;
  formData: any;
  setFormData: (val: any) => void;
  isAdding: boolean;
  setIsAdding: (val: boolean) => void;
}

const initialState: EditSeriesState = {
  currentSeries: null,
  setCurrentSeries: (val: any) => {},
  isEditing: false,
  setIsEditing: (val: boolean) => {},
  currentSelection: null,
  setCurrentSelection: (val: any) => {},
  step: 1,
  setStep: (val: any) => {},
  series: {
    title: '',
    description: '',
  },
  setSeries: (val: any) => {},
  formData: {
    title: '',
    description: '',
    presigned_url: '',
    thumbnail_url: '',
    difficulty: '',
    disciplines: [],
  },
  setFormData: (val: any) => {},
  isAdding: false,
  setIsAdding: (val: boolean) => {},
};

const EditSeriesContext = createContext(initialState);

export const EditSeriesProvider = ({ children }: any) => {
  const [currentSeries, setCurrentSeries] = useState(null);
  const [isEditing, setIsEditing] = useState(false);
  const [currentSelection, setCurrentSelection] = useState<any>(null);
  const [step, setStep] = useState<any>(1);
  const [formData, setFormData] = useState<any>({
    title: '',
    description: '',
    presigned_url: '',
    thumbnail_url: '',
    difficulty: '',
    disciplines: [],
  });
  const [series, setSeries] = useState({
    title: '',
    description: '',
  });
  const [isAdding, setIsAdding] = useState(false);

  return (
    <EditSeriesContext.Provider value={{ 
      currentSeries, 
      setCurrentSeries,
      isEditing,
      setIsEditing,
      currentSelection,
      setCurrentSelection,
      step,
      setStep,
      formData,
      setFormData,
      series,
      setSeries,
      isAdding,
      setIsAdding,
    }}>
      {children}
    </EditSeriesContext.Provider>
  );
};

export const useEditSeriesContext = () => useContext(EditSeriesContext);