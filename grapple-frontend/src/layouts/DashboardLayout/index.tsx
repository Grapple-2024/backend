import Navigation from "@/components/Navigation";
import Sidebar from "@/components/Sidebar";
import { ReactNode, useState } from "react";
import styles from './DashboardLayout.module.css';
import { useSignOut } from "@/hook/auth";
import { useGetUserProfile } from "@/hook/profile";

interface Props {
  sidebarData: any;
  gymName: string;
  children: ReactNode;
  defaultNode: string;
  isStudent?: boolean;
  profile: any;
  gym: any;
  notifications: any;
}

const DashboardLayout = ({ 
  sidebarData,
  gymName,
  children,
  defaultNode, 
  isStudent = false,
  profile: initialProfile,
  gym = null,
  notifications
}: Props) => {  
  const [open, setOpen] = useState(false);
  const mutation = useSignOut();
  const profile = useGetUserProfile(initialProfile);
  
  return (
    <div className={styles.dashboardContainer}>
      <div className={`${open ? styles.sidebarContainerOpen : styles.sidebarContainer}`}>
        <Sidebar 
          profile={profile}
          currentGym={gym}
          title={gymName as string}
          imageSrc={profile?.data?.avatar_url ? profile?.data?.avatar_url : '/placeholder.png'}
          nodes={sidebarData}
          defaultNode={defaultNode}
          line
          open={open}
          onSignOut={() => {
            mutation.mutate();
          }} 
          isCoach={!isStudent}
        />
      </div>
      <div className={styles.mainContainer}>
        <div className={styles.navigationContainer}>
          <>
            <Navigation 
              notifications={notifications}
              gym={gym}
              avatarUrl={profile?.data?.avatar_url ? profile?.data?.avatar_url : '/placeholder.png'}
              open={open}
              isSignedIn={!!profile}
              setOpen={setOpen}
            />
          </>
        </div>
        <div className={styles.content}>
          {children}
        </div>
      </div>
    </div>
  );
};

export default DashboardLayout;
