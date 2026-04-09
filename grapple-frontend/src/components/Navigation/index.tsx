import React, { useContext, useState } from 'react';
import { FaArrowCircleLeft, FaArrowCircleRight, FaBell, FaSearch } from 'react-icons/fa';
import { Dropdown, Image } from "react-bootstrap";
import styles from './navigation.module.css';
import Search from '../Search';
import gymApi from '@/util/gym-api';
import { useRouter } from 'next/navigation';
import { SelectionContext } from './context';
import { useMobileContext } from '@/context/mobile';
import { useToken } from '@/hook/user';
import { searchApi } from '@/hook/base-apis';
import { useGetGym } from '@/hook/gym';

const Navigation = ({ 
  avatarUrl, 
  open,
  setOpen, 
  hasArrow = true,
  isCoach = false,
  isSignedIn = false,
  gym = null,
  notifications = null,
}: any) => {
  const {
    selection,
    setSelection
  } = useContext(SelectionContext);
  
  const [showDropdown, setShowDropdown] = useState(false);
  const router = useRouter();
  const token = useToken();
  const { isMobile } = useMobileContext();
  const [showSearch, setShowSearch] = useState(false);
  
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

  if (isSignedIn.isPending) {
    return null;
  }

  return (
    <div className={styles.navbar}>
      <div>
        {
          hasArrow ? (open ? (
            <FaArrowCircleLeft 
              size={36}
              style={{ cursor: 'pointer' }}
              className={styles.arrowIcon}
              onClick={() => setOpen(!open)}
            />
          ): (
            <FaArrowCircleRight
              size={36}
              style={{ cursor: 'pointer' }}
              onClick={() => setOpen(!open)}
            />
          )): (
            <>
              <Image 
                src="/logo-2.png" 
                onClick={() => {
                  router.push(`/`);
                }}
                style={{
                  cursor: 'pointer',
                  marginRight: 10,
                }}
              />
            </>
          )
        }
      </div>
      {
        !isMobile && (
          <div className={styles.searchContainer}>
            <Search 
              placeholder="Search for a gym or series"
              onChange={(searchOption: any) => {
                setSelection(searchOption.label);
                if (searchOption.type === 'gym') {
                  router.push(`/profile/${searchOption.value}`);
                } else {
                  // Handle series selection
                  router.push(`/${isCoach ? 'coach' : 'student'}/content/${searchOption.value}`);
                }
              }}
              width={500}
              value={selection}
              fetchValues={fetchValues}
            />
          </div>
        )
      }
      <div className={styles.notificationAndAvatar}>
        {
          isMobile && (
            <FaSearch className={styles.searchIcon} onClick={() => setShowSearch(true)} />
          )
        }
        {
          avatarUrl && avatarUrl !== '' && (
            <>
              <Dropdown show={showDropdown} onToggle={() => setShowDropdown(!showDropdown)}>
                <Dropdown.Toggle as="div" id="dropdown-custom-components" className={styles.bellContainer}>
                  <FaBell className={styles.bellIcon} size={24} />
                    <div className={styles.notificationBadge}>{notifications?.length || '0'}</div>
                </Dropdown.Toggle>
                <Dropdown.Menu>
                    {
                      notifications?.length === 0 ? <Dropdown.Item>No new notifications</Dropdown.Item> : notifications?.map((req: any) => (
                        <Dropdown.Item href="#/action-1" key={req.id}>
                          <span onClick={() => {
                            router.push(`/coach/users`);
                          }}>
                            {req.first_name} {req.last_name} has requested to join your gym
                          </span>
                        </Dropdown.Item>
                      ))
                    }
                </Dropdown.Menu>
              </Dropdown>
              <Image 
                src={avatarUrl}
                style={{ 
                  objectFit: 'cover',
                  width: 50,
                  height: 50,
                  clipPath: 'circle()',
                  cursor: 'pointer',
                }} 
                alt="User Avatar"
                onClick={() => {
                  router.push(`/${isCoach ? 'coach' : 'student'}/settings`);
                }}
              />
            </>
          )
        }
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

export default Navigation;
