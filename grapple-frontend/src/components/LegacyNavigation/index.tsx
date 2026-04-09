import { colors } from '@/util/colors';
import { useRouter } from 'next/navigation';
import { Button, Image } from 'react-bootstrap';
import Container from 'react-bootstrap/Container';
import Nav from 'react-bootstrap/Nav';
import Navbar from 'react-bootstrap/Navbar';
import NavDropdown from 'react-bootstrap/NavDropdown';
import { useContext } from 'react';
import { useMobileContext } from '@/context/mobile';
import { SelectionContext } from '../Navigation/context';
import Search from '../Search';
import gymApi from '@/util/gym-api';
import { useUserContext } from '@/context/user';
import { useGetUserProfile } from '@/hook/profile';
import { useIsSignedIn, useToken } from '@/hook/user';
import { searchApi } from '@/hook/base-apis';
import { useGetGym } from '@/hook/gym';
interface NavigationProps {
  onSignOut: () => void;
};

const LegacyNavigation = ({ onSignOut }: NavigationProps) => {
  const router = useRouter();

  const profile = useGetUserProfile();

  const { isMobile } = useMobileContext();
  
  const isSignedIn = useIsSignedIn(); 
  const token = useToken();
  const gymData = useGetGym();

  const gym = gymData?.data;

  const userName = profile?.data?.first_name as string | null;

  const { 
    selection, 
    setSelection,
  } = useContext(SelectionContext);

  const { role } = useUserContext();
  const isCoach = role === 'Owner' || role === 'Coach';

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
    <Navbar 
      collapseOnSelect 
      expand="xl" 
      style={{
        backgroundColor: colors.black,
        color: colors.white,
      }}
      variant="dark"
    >
      <Container>
        <Navbar.Brand onClick={() => {
          router.push('/');
        }}>
          <Image alt="Gym Logo Image" src='/logo.png' style={{ objectFit: 'cover', height: '5vh'  }}/>
        </Navbar.Brand>
        <Navbar.Toggle aria-controls="responsive-navbar-nav" />
        <Navbar.Collapse id="responsive-navbar-nav">
          {
            (
              <Nav className="me-auto">
                {
                  (isSignedIn && !profile?.isPending) && (
                    <Nav.Link
                      className='nav-link'
                      style={{ color: `${colors.white} !important` }}
                      onClick={() => {
                        if (isCoach) {
                          router.push(`/coach/my-gym`);
                        } else {
                          router.push(`/student/my-gym`);
                        }
                      }}
                    >
                      <div style={{ color: `${colors.white} !important` }}>My Gym</div>
                    </Nav.Link>
                  )
                }
                {
                  (isMobile && isSignedIn && !profile?.isPending) && (
                    <Nav.Link 
                      onClick={() => {
                        if (isCoach) {
                          router.push(`/coach/settings`);
                        } else {
                          router.push(`/studentsettings`);
                        }
                      }}
                    >
                      Settings
                    </Nav.Link>
                  )
                }
              </Nav>
            )
          }
          {
            <Nav>
              {
                !isMobile && (
                  <div style={{
                    marginRight: isMobile ?  0 : 30,
                    marginTop: 'auto',
                    marginBottom: 'auto',
                    zIndex: 1000,
                  }}>
                    
                    <Search 
                      placeholder="Search for a gym"
                      onChange={(searchOption: any) => {
                        setSelection(searchOption.label);
                        router.push(`/profile/${searchOption.value}`);
                      }}
                      width={500}
                      value={selection}
                      fetchValues={fetchValues as any}
                    />
                  </div>
                )
              }
            </Nav>
          }
          {
            <Nav>
              {
                (!isSignedIn) ? (
                  <>
                    {
                      isMobile ? (
                        <Nav.Link onClick={() => {
                          router.push('/auth');
                        }}>
                          Login
                        </Nav.Link>
                      ): (
                        <Nav.Link style={{ color: `${colors.white} !important` }} onClick={() => {
                          router.push('/auth');
                        }}>
                          Login
                        </Nav.Link>
                      )
                    }
                  </>
                ) : (
                  <>
                    {
                      userName && !isMobile && !profile?.isPending && (
                        <Navbar.Collapse className="justify-content-end" style={{
                          marginRight: 20
                        }}>
                          <Navbar.Text style={{ color: `${colors.white} !important` }}>
                            Hello, {userName}
                          </Navbar.Text>
                        </Navbar.Collapse>
                      )
                    }
                    {
                      (isSignedIn && !profile?.isPending) && (
                        <Nav.Link>
                          <Button 
                            style={{ 
                              backgroundColor: colors.primary, 
                              borderColor: colors.primary,
                              color: colors.black,
                            }} 
                            onClick={onSignOut}
                          >
                            Sign Out
                          </Button>
                        </Nav.Link>
                      )
                    }
                  </>
                )
              }
            </Nav>
          }
        </Navbar.Collapse>
        {
          isMobile && (
            <div style={{
              width: isMobile ? 300 : 400,
              margin: 'auto',
              marginTop: 20,
              zIndex: 1000,
            }}>
              <Search 
                placeholder="Search for a gym"
                onChange={(searchOption: any) => {
                  setSelection(searchOption.label);
                  router.push(`/profile/${searchOption.value}`);
                }}
                width={300}
                value={selection}
                fetchValues={fetchValues as any}
              />
            </div>
          )
        }
      </Container>
    </Navbar>
  );
};

export default LegacyNavigation;