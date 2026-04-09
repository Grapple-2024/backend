import React from "react";
import { Col, Row } from "react-bootstrap";
import 'react-datepicker/dist/react-datepicker.css';
import './styles.css'; // Custom styles to match your design
import ProfileSchedule from "@/components/Profile/Display/Schedule";
import Announcements from "@/components/Announcements";
import DatePicker from "@/components/DatePicker";
import DashboardTabs from "@/components/DashboardTabs";
import TechniquesOfTheWeek from "@/components/TechniquesOfTheWeek";
import Upgrade from "@/components/Upgrade";

const days = [
  'Sun',
  'Mon', 
  'Tue', 
  'Wed', 
  'Thu', 
  'Fri', 
  'Sat'
];

interface StudentDashboardDesktopProps {
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

const StudentDashboardDesktop = ({
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
}: StudentDashboardDesktopProps) => {
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
          data={announcementData} 
          coachAvatar={announcementData?.length > 0 ? announcementData[0].coach_avatar : '/placeholder.png'}
          coachName={announcementData?.length > 0 ? announcementData[0].coach_name : ''}
        />
      )
    },
  ];
  
  return (
    <Row style={{ margin: 0, padding: 0 }}>
      <Col xs={8} style={{ margin: 0, padding: 0, height: '92vh', overflow: 'auto' }}>
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
      </Col>
      <Col xs={4} style={{ borderLeft: '2px solid #CBD5E0', padding: 0, margin: 0, height: '92vh' }}>
        <Row style={{ margin: 0, paddingRight: 30, paddingLeft: 30, paddingTop: 10, height: '16vh' }}>
          <DatePicker onDaySelect={(day: any) => {
            updateAnnouncements.mutate({ date: day })
            updateTechniques.mutate({ date: day });
            
            setDaySelected(day);
          }} />
        </Row>
        <Row style={{ margin: 0, paddingRight: 30, paddingLeft: 30, paddingTop: 30, height: '48vh' }}>
          <ProfileSchedule 
            schedule={gym?.data?.schedule}
            days={days}
            selectedDay={getCurrentDay(daySelected)}
            daily
          />
        </Row>
        <Row style={{ margin: 0, padding: '0px 30px 0px 30px', height: '22vh', marginTop: 10 }}>
          <Upgrade />
        </Row>
      </Col>
    </Row>
  );
};

export default StudentDashboardDesktop;
