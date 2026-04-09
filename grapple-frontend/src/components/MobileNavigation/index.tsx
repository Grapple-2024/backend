import React, { useContext, useState } from 'react';
import { FaSearch, FaBell } from 'react-icons/fa';
import { Dropdown, Image } from "react-bootstrap";
import { useRouter } from 'next/navigation';
import { SelectionContext } from '../Navigation/context';
import gymApi from '@/util/gym-api';
import styles from './MobileNavigation.module.css';
import Search from '../Search';
import { useUserContext } from '@/context/user';
import { useIsSignedIn, useToken } from '@/hook/user';
import { searchApi } from '@/hook/base-apis';
import { useGetGym } from '@/hook/gym';

const MobileNavigation = ({ avatarUrl, notifications }: any) => {
  const router = useRouter();
  const [showDropdown, setShowDropdown] = useState(false);
  const [showSearch, setShowSearch] = useState(false);
  const isSignedIn = useIsSignedIn();
  const token = useToken();
  const { 
    selection, 
    setSelection,
  } = useContext(SelectionContext);
  const {
    role
  } = useUserContext();
  const gymData = useGetGym();

  const gym = gymData?.data;
  
  const isCoach = role === 'Owner';
  
  const fetchValues = async (inputValue: string) => {
    if (!inputValue || inputValue.length === 0) {
      return [];
    }

    let values;
    
    if (isSignedIn) {
      const { data: { data: gymData } } = await gymApi.get<any>('/gyms', {
        params: {
          name: inputValue,
        },
        headers: {
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        }
      });
      
      const { data } = await searchApi.get<any>('', {
        params: {
          query: inputValue,
          ...(gym?.id ? { gym_id: gym?.id } : {}),
          gym_id: gym?.id,
        },
        headers: {
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        }
      });
    
      values = {
        gyms: gymData,
        series: data?.series,
      };
    } else {
      const { data: { data } } = await gymApi.get<any>('/gyms', {
        params: {
          name: inputValue,
        },
        headers: {
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        }
      });
      
      values = {
        gyms: data,
        series: [],
      };
    }
    
    const gymOptions = values?.gyms ? values.gyms.map((gym: any) => ({
      label: gym.name,
      value: gym.id,
      type: 'gym' as const,
    })) : [{ label: "No gyms found", value: "" }];
  
    const seriesOptions = values.series ? values.series.map((series: any) => ({
      label: series.title,
      value: series.id,
      type: 'series' as const,
    })) : [{ label: "No series found", value: "" }];
    
    return [
      { label: 'Gyms', options: gymOptions },
      { label: 'Series', options: seriesOptions },
    ];
  };
  
  return (
    <div className={styles.navbar}>
      <div className={styles.navbarTop}>
        {!showSearch && (
          <>
            <div className={styles.burgerPlaceholder}></div>
            <div className={styles.logoContainer}>
              <Image src="/logo-2.png" alt="Grapple Logo" className={styles.logo} onClick={() => {
                router.push('/');
              }}/>
            </div>
            <div className={styles.notificationAndAvatar}>
              <FaSearch className={styles.searchIcon} onClick={() => setShowSearch(true)} />
              {
                isSignedIn && (
                  <>
                    <Dropdown show={showDropdown} onToggle={() => setShowDropdown(!showDropdown)}>
                      <Dropdown.Toggle as="div" id="dropdown-custom-components" className={styles.bellContainer}>
                        <FaBell className={styles.bellIcon} size={24} />
                        <div className={styles.notificationBadge}>0</div>
                      </Dropdown.Toggle>
                      <Dropdown.Menu>
                        <Dropdown.Item href="#/action-1">Coming Soon...</Dropdown.Item>
                      </Dropdown.Menu>
                    </Dropdown>
                    <Image 
                      src={avatarUrl}
                      className={styles.avatar}
                      alt="User Avatar"
                    />
                  </>
                )
              }
            </div>
          </>
        )}

        {showSearch && (
          <div className={styles.searchContainer}>
            <Search 
              placeholder="Search for a gym"
              onChange={(searchOption: any) => {
                setSelection(searchOption.label);
                if (searchOption.type === 'gym') {
                  router.push(`/profile/${searchOption.value}`);
                } else {
                  // Handle series selection
                  router.push(`/${isCoach ? 'coach' : 'student'}/content/${searchOption.value}`);
                }
              }}
              width={275}
              value={selection}
              fetchValues={fetchValues as any}
            />
            <button className={styles.cancelButton} onClick={() => setShowSearch(false)}>
              Cancel
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default MobileNavigation;
