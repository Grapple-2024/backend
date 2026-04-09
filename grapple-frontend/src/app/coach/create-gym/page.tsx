'use client';

import Select from 'react-select';
import styles from './CreateGym.module.css';
import Scheduler from "./components/Scheduler";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useMobileContext } from '@/context/mobile';
import { FaArrowRight, FaSearch } from 'react-icons/fa';
import { Button } from 'react-bootstrap';
import axios from 'axios';
import { useCreateGym } from '@/hook/gym';
import { useGetUserProfile } from '@/hook/profile';
import { useToken } from '@/hook/user';

interface MapboxFeature {
  place_name: string;
  context: Array<{
    id: string;
    text: string;
    short_code?: string;
  }>;
  geometry: {
    coordinates: [number, number]
  };
  properties: {
    full_address: string;
  };
  text: string;
}

const selections = [
  { value: "jiu-jitsu", label: "Jiu-Jitsu" },
  { value: "boxing", label: "Boxing" },
  { value: "wrestling", label: "Wrestling" },
  { value: "mma", label: "MMA" },
  { value: "karate", label: "Karate" },
  { value: "muay-thai", label: "Muay Thai" },
  { value: "judo", label: "Judo" },
  { value: "taekwondo", label: "Taekwondo" }
];

const scheduleInitial = {
  'sun': [],
  'mon': [],
  'tue': [],
  'wed': [],
  'thu': [],
  'fri': [],
  'sat': []
};

const API = process.env.NEXT_PUBLIC_API_HOST;

const CreateGym = () => {  
  const { isMobile } = useMobileContext();
  const router = useRouter();

  const [formData, setFormData] = useState({
    name: "",
    address_line_1: "",
    address_line_2: "",
    city: "",
    state: "",
    zip: "",
    country: "USA",
    disciplines: [],
    creator: "",
    longitude: '0',
    latitude: '0',
    schedule: scheduleInitial,
    description: "",
    banner_image: "",
    logo_image: "",
  });

  const [addressSearch, setAddressSearch] = useState("");
  const [addressResults, setAddressResults] = useState<MapboxFeature[]>([]);
  const [showResults, setShowResults] = useState(false);
  const [isNextAvailable, setIsNextAvailable] = useState(true);
  const token = useToken();
  const profile = useGetUserProfile();

  useEffect(() => {
    if (
      formData.address_line_1 !== '' ||
      formData.name !== ''
    ) {
      setIsNextAvailable(false);
    } else {
      setIsNextAvailable(true);
    }
  }, [formData]);

  const searchAddress = async (query: string) => {
    if (!query || query.length < 3) {
      setAddressResults([]);
      return;
    }

    try {
      const { data } = await axios.get(
        `${API}/mapbox?q=${query}`,
        {
          headers: {
            Authorization: `Bearer ${token}`
          }
        }
      );
      
      setAddressResults(data.features);
      
      setShowResults(true);
    } catch (error) {
      console.error('Error searching address:', error);
    }
  };

  const handleAddressSelect = (feature: MapboxFeature) => {
    const addressParts = feature.properties.full_address.split(',');
    const streetAddress = addressParts[0].trim();
    const city = addressParts[1]?.trim() || '';
    const stateAndZip = addressParts[2]?.trim().split(' ') || [];
    const state = stateAndZip[0] || '';
    const zip = stateAndZip[1] || '';

    setFormData({
      ...formData,
      address_line_1: streetAddress,
      city,
      state,
      zip,
      longitude: feature.geometry.coordinates[0].toString(),
      latitude: feature.geometry.coordinates[1].toString()
    });
    
    setAddressSearch(feature.properties.full_address);
    setShowResults(false);
  };

  const mutation = useCreateGym();
  
  const setSchedule = (newSchedule: any[]) => {
    setFormData({ ...formData, schedule: newSchedule } as any);
  }

  return (
    <div className={styles.container}>
      <div className={styles.formSection}>
        <div className={`${styles.formColumn} ${isMobile ? styles.mobileColumn : ''}`}>
          <h4 className={styles.sectionTitle}>Account Details</h4>
          
          <div className={styles.inputContainer}>
            <input
              type="text"
              placeholder="Gym Name"
              value={formData.name}
              onChange={e => setFormData({ ...formData, name: e.target.value })}
              className={styles.input}
            />
          </div>

          <div className={styles.addressSearchContainer}>
            <div className={styles.searchInputWrapper}>
              <input
                type="text"
                placeholder="Search address..."
                value={addressSearch}
                onChange={(e) => {
                  setAddressSearch(e.target.value);
                  searchAddress(e.target.value);
                }}
                className={styles.searchInput}
              />
              <FaSearch className={styles.searchIcon} />
            </div>
            
            {showResults && addressResults?.length > 0 && (
              <div className={styles.searchResults}>
                {addressResults.map((result, index) => (
                  <div
                    key={index}
                    className={styles.searchResult}
                    onClick={() => handleAddressSelect(result)}
                  >
                    {result?.properties?.full_address}
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className={styles.addressDetails}>
            <div className={styles.addressRow}>
              <input
                type="text"
                placeholder="Address 2 (Optional)"
                value={formData.address_line_2}
                onChange={e => setFormData({ ...formData, address_line_2: e.target.value })}
                className={styles.input}
              />
            </div>
          </div>

          <div className={styles.textareaContainer}>
            <label className={styles.label}>Gym Description</label>
            <textarea
              placeholder="Give a brief description of your gym"
              rows={3}
              value={formData.description}
              onChange={e => setFormData({ ...formData, description: e.target.value })}
              className={styles.textarea}
            />
          </div>

          <h4 className={styles.sectionTitle}>Disciplines</h4>
          <div className={styles.disciplineRow}>
            <div style={{ width: '60%' }}>
              <Select
                isMulti
                name="Disciplines"
                options={selections}
                classNamePrefix="select"
                onChange={e => {
                  setFormData({
                    ...formData,
                    disciplines: e.map((discipline: any) => discipline.value) as any
                  });
                }}
              />
            </div>
          </div>
        </div>

        <div className={`${styles.scheduleColumn} ${isMobile ? styles.mobileColumn : ''}`}>
          <h4 className={styles.sectionTitle}>Schedule</h4>
          <Scheduler schedule={formData.schedule} setSchedule={setSchedule}/>
        </div>
      </div>

      <div className={styles.navigationButtons}>
        <Button 
          className={styles.skipButton}
          onClick={() => router.push('/student/my-gym')}
        >
          Skip
        </Button>
        <Button 
          className={styles.nextButton}
          onClick={() => mutation.mutate({
            ...formData,
            coach_first_name: profile?.data?.first_name,
            coach_last_name: profile?.data?.last_name,
          } as any)}
          disabled={isNextAvailable}
        >
          <FaArrowRight size={24} />
        </Button>
      </div>
    </div>
  );
};

export default CreateGym;