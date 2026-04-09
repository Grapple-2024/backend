import { Image } from 'react-bootstrap';
import styles from './LogoSection.module.css';
import { CiEdit } from 'react-icons/ci';
import { FaSearch } from 'react-icons/fa';
import { useState } from 'react';
import axios from 'axios';
import { useToken } from '@/hook/user';

interface LogoSectionProps {
  isCoach?: boolean;
  logoUrl: string;
  setEditingImage?: (imageType: string) => void;
  setIsModalOpen?: (isOpen: boolean) => void;
  updateGym?: any;
  gym: any;
};

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

const API = process.env.NEXT_PUBLIC_API_HOST;

const LogoSection = ({
  isCoach = false,
  logoUrl,
  setEditingImage,
  setIsModalOpen,
  updateGym,
  gym
}: LogoSectionProps) => {
  const [isUpdatingInfo, setIsUpdatingInfo] = useState(false);
  const [newGymName, setNewGymName] = useState('');
  const [addressSearch, setAddressSearch] = useState('');
  const [addressResults, setAddressResults] = useState<any[]>([]);
  const [showResults, setShowResults] = useState(false);
  const [newAddress, setNewAddress] = useState<{
    address_line_1: string;
    city: string;
    state: string;
    zip: string;
    longitude?: string;
    latitude?: string;
  } | null>(null);
  const token = useToken();

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

    setNewAddress({
      ...newAddress,
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
  
  if (isCoach) {
    return (
      <div className={styles.logoSection}>
        <div className={styles.logoImageContainer}>
          <Image 
            src={logoUrl || "/placeholder-logo.jpeg"} 
            alt="Gym Logo"
            width={100}
            height={100}
            className={styles.logoImage}
          />
          <div className={styles.logoEditButton} onClick={() => {
            setEditingImage && setEditingImage('logo');
            setIsModalOpen && setIsModalOpen(true);
          }}>
            <CiEdit color="white" size={20}/>
          </div>
        </div>
        <div className={styles.gymInfoSection}>
          {isUpdatingInfo ? (
            <div className={styles.editInfoContainer}>
              <input
                type="text"
                value={newGymName}
                onChange={(e) => setNewGymName(e.target.value)}
                className={styles.editInput}
                placeholder="Gym Name"
              />
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
                
                {showResults && addressResults.length > 0 && (
                  <div className={styles.searchResults}>
                    {addressResults.map((result: any, index: number) => (
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
              <div className={styles.editActions}>
                <button 
                  className={styles.cancelButton}
                  onClick={() => {
                    setIsUpdatingInfo(false);
                    setAddressSearch('');
                    setShowResults(false);
                  }}
                >
                  Cancel
                </button>
                <button 
                  className={styles.saveButton}
                  onClick={() => {
                    updateGym.mutate({
                      ...gym,
                      name: newGymName,
                      address_line_1: newAddress?.address_line_1,
                      city: newAddress?.city,
                      state: newAddress?.state,
                      zip: newAddress?.zip,
                    });
                    setIsUpdatingInfo(false);
                    setAddressSearch('');
                    setShowResults(false);
                  }}
                >
                  Save
                </button>
              </div>
            </div>
          ) : (
            <>
              <div className={styles.gymNameSection}>
                <h1 className={styles.gymName}>{gym?.name}</h1>
                <CiEdit 
                  size={20} 
                  className={styles.editIcon}
                  onClick={() => {
                    setIsUpdatingInfo(true);
                    setNewGymName(gym?.name);
                    setAddressSearch(`${gym?.address_line_1}${gym?.address_line_2 ? ', ' + gym?.address_line_2 : ''}, ${gym?.city}, ${gym?.state}`);
                  }}
                />
              </div>
              <div className={styles.locationInfo}>
                <span>{`${gym?.address_line_1}${gym?.address_line_2 ? ', ' + gym?.address_line_2 : ''}`}</span>
                <br />
                <span>{`${gym?.city}, ${gym?.state}`}</span>
              </div>
            </>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className={styles.logoSection}>
      <div className={styles.logoImageContainer}>
        <Image 
          src={logoUrl || "/placeholder-logo.jpeg"} 
          alt="Gym Logo"
          width={100}
          height={100}
          className={styles.logoImage}
        />
      </div>
      <div className={styles.gymInfoSection}>
        <div className={styles.gymNameSection}>
          <h1 className={styles.gymName}>{gym?.name}</h1>
        </div>
        <div className={styles.locationInfo}>
          <span>{`${gym?.address_line_1}${gym?.address_line_2 ? ', ' + gym?.address_line_2 : ''}`}</span>
          <br />
          <span>{`${gym?.city}, ${gym?.state}`}</span>
        </div>
      </div>
    </div>
  );
};

export default LogoSection;