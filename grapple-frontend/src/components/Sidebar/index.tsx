import styles from './sidebar.module.css';
import { useState } from 'react';
import { Row, Col, Image } from 'react-bootstrap';
import { useRouter } from 'next/navigation';
import { FaChevronDown, FaChevronUp, FaPlus } from 'react-icons/fa';
import { useMobileContext } from '@/context/mobile';
import GrappleIcon from '../GrappleIcon';
import NodeRow from './NodeRow';
import { useChangeGym, useGetGym } from '@/hook/gym';
import { useGetUserProfile } from '@/hook/profile';

interface LinkNode {
  title: string;
  route?: string;
  line: boolean;
  cb?: () => void;
  href?: string;
  active?: boolean;
  Icon?: React.ReactNode;
}

interface SidebarProps {
  title: string;
  imageSrc: string;
  line?: boolean;
  nodes: LinkNode[];
  defaultNode: string;
  open?: boolean;
  onSignOut: () => void;
  onClose?: () => void;
  currentGym: any;  
  profile: any;
  isCoach?: boolean;
}

const Sidebar = ({
  title,
  imageSrc,
  line = false,
  defaultNode,
  nodes,
  open = false,
  onSignOut,
  onClose,
  isCoach = false,
}: SidebarProps) => {
  const router = useRouter();
  const [active, setActive] = useState<string>(defaultNode);
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const { isMobile } = useMobileContext();
  const profile = useGetUserProfile();
  const currentGym = useGetGym();
  const setCurrentGym = useChangeGym();
  const gyms = profile?.data?.gyms;
  
  const isActive = (field: string) => {
    setActive(field);
  };

  const handleAvatarRowClick = () => {
    setDropdownOpen(!dropdownOpen);
  };
  
  return (
    <div className={`${styles.sidebarContainer} ${open ? styles.sidebarOpen : styles.sidebarClosed}`}>
      <Row 
        className={`${styles.avatarRow} ${dropdownOpen ? styles.dropdownOpen : ''}`} 
        onClick={handleAvatarRowClick}
      >
        {open ? (
          <>
            {
              profile?.isPending ? null : (
              <div className={styles.gymSelectorContainer}>
                <Col xs="auto" className={styles.logoColumn}>
                  <Image 
                    src={currentGym?.data?.logo_url || "/logo-2.png"} 
                    width={50}
                    height={50}
                    style={{
                      cursor: 'pointer',
                      objectFit: 'contain',
                    }}
                  />
                </Col>
                <Col className={styles.gymNameColumn}>
                  <span className={styles.gymName}>
                    {!currentGym?.data?.name && "Join or create a gym"}
                    {currentGym?.data && currentGym?.data?.name?.length > 16
                      ? `${currentGym?.data?.name.substring(0, 16)}...` 
                      : currentGym?.data?.name}
                  </span>
                </Col>
                <Col xs="auto" className={styles.dropdownIconColumn}>
                  {dropdownOpen ? <FaChevronUp /> : <FaChevronDown />}
                </Col>
              </div>
              )
            }
            {(dropdownOpen && !profile?.isPending) && (
              <div className={styles.dropdownContainer}>
                {!gyms ? (
                  <div 
                    className={styles.createGymItem}
                    onClick={(e) => {
                      e.stopPropagation();
                      router.push('/coach/create-gym');
                      setDropdownOpen(false);
                    }}
                  >
                    <span>Create a new gym</span>
                    <FaPlus size={14} />
                  </div>
                ) : (
                  <>
                    {gyms?.map((data: any) => (
                      <div 
                        key={data?.gym?.id} 
                        className={styles.dropdownItem}
                        onClick={(e) => {
                          e.stopPropagation();
                          setCurrentGym.mutate({ gym: data?.gym, group: data?.group });
                          setDropdownOpen(false);
                        }}
                      >
                        {data?.gym?.name}
                      </div>
                    ))}
                    <div 
                      className={styles.createGymItem}
                      onClick={(e) => {
                        e.stopPropagation();
                        router.push('/coach/create-gym');
                        setDropdownOpen(false);
                      }}
                    >
                      <span>Create a new gym</span>
                      <FaPlus size={14} />
                    </div>
                  </>
                )}
              </div>
            )}
          </>
        ) : (
          <>
            <Col xs="auto" className={!open && isMobile ? styles.avatarColumnCollapsed : styles.avatarColumn}>
              <Image 
                src={currentGym?.data?.logo_url || '/mini-logo.png'}
                style={{ height: '7vh'}}
              /> 
            </Col>
          </>
        )}
      </Row>
      <NodeRow 
        nodes={nodes}
        active={active}
        open={open}
        isActive={isActive}
        router={router}
        isCoach={isCoach}
      />
      {nodes &&
        nodes.map((node, index) => {
          if (node.line) {
            return (
              <Row
                key={index}
                className={`${styles.navLinkSidebar} ${active === node.title ? styles.navLinkSidebarActive : ''}`}
                onClick={() => {
                  router.push(`/${isCoach ? 'coach' : 'student'}/${node.route}`);
                }}
              >
                <div className={styles.navLinkTitle}>{node.title}</div>
              </Row>
            );
          }
        })}
      <div className={open ? styles.signOutContainerOpen : styles.signOutContainerClosed} onClick={onSignOut}>
        {
          !open ? (
            <GrappleIcon src='/logout.svg' variant='dark'/>
          ) : (
            <div className={styles.signOutTextContainer}>
              <GrappleIcon src='/logout.svg' variant='dark'/>
              <div className={styles.signOutText}>Sign Out</div>
            </div>
          )
        }
      </div>
    </div>
  );
};

export default Sidebar;