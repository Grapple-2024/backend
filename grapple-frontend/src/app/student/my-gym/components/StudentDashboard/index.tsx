import { useMobileContext } from "@/context/mobile";
import StudentDashboardMobile from "./StudentDashboardMobile";
import StudentDashboardDesktop from "./StudentDashboardDesktop";
import { useEffect, useState } from "react";
import { useGetGym } from "@/hook/gym";
import { useGetAnnouncements, useUpdateAnnouncements } from "@/hook/announcement";
import { useGetTechniques, useUpdateTechniques } from "@/hook/techniques";
import { useGetByRequestor, useGetStudentsRequest } from "@/hook/request";
import PageSkeleton from "@/components/PageSkeleton";


const StudentDashboard = ({
  intialAnnouncements, 
  isAcceptedUser: accepted
}: any) => {
  const { isMobile } = useMobileContext();
  const [daySelected, setDaySelected] = useState<Date>(new Date());
  const [isAcceptedUser, setIsAcceptedUser] = useState<boolean>(false);

  const gym = useGetGym();
  const announcements = useGetAnnouncements(intialAnnouncements);
  const techniques = useGetTechniques();
  const updateAnnouncements = useUpdateAnnouncements();
  const updateTechniques = useUpdateTechniques();
  const requests = useGetStudentsRequest();
  
  useEffect(() => {
    setIsAcceptedUser(accepted);
  }, []);
  
  function getCurrentDay(daySelected: Date): string {
    const daysOfWeek = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday', ];
    const currentDayIndex = daySelected.getDay();
    
    return daysOfWeek[currentDayIndex - 1 === -1 ? 6 : currentDayIndex - 1];
  }
  
  if (requests.isPending) {
    return <PageSkeleton />;
  }

  const requestData = requests?.data?.[0];

  return (
    <>
      {
        isMobile ? (
          <StudentDashboardMobile 
            techniques={techniques}
            announcementData={announcements?.data}
            gym={gym}
            getCurrentDay={getCurrentDay}
            daySelected={daySelected}
            setDaySelected={setDaySelected}
            isAcceptedUser={isAcceptedUser}
            updateAnnouncements={updateAnnouncements}
            updateTechniques={updateTechniques}
            requestStatus={requestData?.status}
          />
        ) : (
          <StudentDashboardDesktop 
            techniques={techniques}
            announcementData={announcements?.data}
            gym={gym}
            getCurrentDay={getCurrentDay}
            daySelected={daySelected}
            setDaySelected={setDaySelected}
            isAcceptedUser={isAcceptedUser}
            updateAnnouncements={updateAnnouncements}
            updateTechniques={updateTechniques}
            requestStatus={requestData?.status}
          />
        )
      }
    </>
  );
};

export default StudentDashboard;