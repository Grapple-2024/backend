import React from "react";
import { Row } from "react-bootstrap";
import 'react-datepicker/dist/react-datepicker.css';
import ProfileSchedule from "@/components/Profile/Display/Schedule";
import Announcements from "@/components/Announcements";
import DatePicker from "@/components/DatePicker";
import TechniquesOfTheWeek from "@/components/TechniquesOfTheWeek";
import DashboardTabs from "@/components/DashboardTabs";

const days = [
  'Sun',
  'Mon', 
  'Tue', 
  'Wed', 
  'Thu', 
  'Fri', 
  'Sat'
];

interface StudentDashboardMobileProps {
  techniques: any;
  announcementData: any;
  gym: any;
  getCurrentDay: any;
  daySelected: Date;
  isAcceptedUser: boolean;
  setDaySelected: any;
  updateAnnouncements: any;
  updateTechniques: any;
  requestStatus: string | null;
};

const StudentDashboardMobile = ({
  techniques, 
  announcementData,
  gym,
  getCurrentDay,
  daySelected,
  isAcceptedUser,
  setDaySelected,
  updateAnnouncements,
  updateTechniques,
  requestStatus,
}: StudentDashboardMobileProps) => {
  

  const tabs = [
    {
      label: 'Techniques of the Week',
      content: (
        <TechniquesOfTheWeek 
          series={techniques?.data}
          create={false}
          path='/student/content'
        />
      )
    },
    {
      label: 'Announcements',
      content: (
        <Announcements 
          create={false}
          data={announcementData?.data} 
          coachAvatar='/placeholder.png'
          coachName="Adam Watts"
        />
      )
    },
    {
      label: 'Schedule',
      content: (
        <Row style={{ borderLeft: '2px solid #CBD5E0', padding: 0, margin: 0, height: 'auto' }}>
          <ProfileSchedule 
            schedule={gym?.data?.schedule}
            days={days}
            selectedDay={getCurrentDay(daySelected)}
            daily
          />
        </Row>
      )
    }
  ];

  return (
    <>
      <Row style={{ margin: 0, padding: 0 }}>
        <Row style={{ marginBottom: 50, padding: '0 20px 0 20px', maxWidth: '100vw' }}>
          <DatePicker 
            onDaySelect={(day) => {
              updateAnnouncements.mutate({ date: day });
              updateTechniques.mutate({ date: day });
              setDaySelected(day);
            }}
          />
        </Row>
        <Row style={{ margin: 0, padding: 0 }}>
          {
            isAcceptedUser ? (
              <DashboardTabs 
                tabs={tabs}
              />
            ): (
              <div style={{ 
                display: 'flex',
                textAlign: 'center',
                justifyContent: 'center',
                alignItems: 'center',
                height: '100%',
                padding: 20,
                margin: 0,
                fontSize: 32,
                fontWeight: 'bold' 
              }}>
              {
                requestStatus === 'Pending' ? (
                  <>
                    Your gym request is pending, please wait to be accepted
                  </>
                ) : (
                  <>
                    You are currently not apart of any gyms, please use the search bar at the top to find a gym and request access
                  </>
                )
              }
              </div>
            )
          }
        </Row>
      </Row>
  </>
  );
};

export default StudentDashboardMobile;
