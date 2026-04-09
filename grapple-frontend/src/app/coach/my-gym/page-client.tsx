"use client";

import styles from './CoachDashboard.module.css';

import { Col, Row } from "react-bootstrap";
import DatePicker from "@/components/DatePicker";
import DashboardTabs from "@/components/DashboardTabs";
import ProfileSchedule from "@/components/Profile/Display/Schedule";
import Upgrade from "@/components/Upgrade";
import TechniquesOfTheWeek from '@/components/TechniquesOfTheWeek';
import Announcements from '@/components/Announcements';
import { useEffect, useState } from "react";
import PageSkeleton from "@/components/PageSkeleton";
import { useRouter } from "next/navigation";
import { useMobileContext } from "@/context/mobile";
import { useGetGym } from '@/hook/gym';
import { GymSchedule } from '@/types/gym';
import { useCreateAnnouncement, useDeleteAnnouncement, useGetAnnouncements, useUpdateAnnouncements } from '@/hook/announcement';
import { useCreateTechnique, useDeleteTechnique, useGetTechniques, useUpdateTechniques } from '@/hook/techniques';
import { useGetUserProfile } from '@/hook/profile';

const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

const MyGymPage = ({ announcements }: any) => {
  const { isMobile } = useMobileContext();  
  const router = useRouter();
  const [daySelected, setDaySelected] = useState<Date>(new Date());
  const [overrideTab, setOverrideTab] = useState<any>(undefined);

  const gym = useGetGym();
  const gymData = gym?.data;
  
  const announcementsQuery = useGetAnnouncements(announcements, daySelected);
  const updateGymAnnouncements = useUpdateAnnouncements();
  const profile = useGetUserProfile();
  const announcementData = announcementsQuery?.data;
  
  const createAnnouncement = useCreateAnnouncement();
  const deleteAnnouncement = useDeleteAnnouncement();  

  const techniques = useGetTechniques();

  const createTechnique = useCreateTechnique();
  const updateTechniques = useUpdateTechniques();

  const deleteTechniques = useDeleteTechnique();
  
  function getCurrentDay(daySelected: Date): string {
    const daysOfWeek = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday', ];
    const currentDayIndex = daySelected.getDay();
    
    return daysOfWeek[currentDayIndex - 1 === -1 ? 6 : currentDayIndex - 1];
  }
  
  useEffect(() => {
    if (!gymData && !gym.isPending) {
      router.push(`/coach/create-gym`);
    }
  }, [gymData, gym?.isPending]);
  
  if (gym?.isPending || announcementsQuery?.isPending || gym?.isPending || techniques?.isPending) {
    return <PageSkeleton />;
  }
  const tabs = [
    {
      label: 'Techniques of the Week',
      content: (
        <TechniquesOfTheWeek 
          quickAction={overrideTab === 0}
          series={techniques?.data}
          onSave={createTechnique as any}
          onDelete={deleteTechniques.mutate}
          create={true}
          daySelected={daySelected}
        />
      )
    },
    {
      label: 'Announcements',
      content: (
        <Announcements 
          quickAction={overrideTab === 1}
          create={true}
          onDelete={(id) => deleteAnnouncement.mutate(id)}
          onSave={(announcement: any) => {
            createAnnouncement.mutate({
              ...announcement,
              gym_id: gymData?.id,
            });
          }}
          isCoach
          data={announcementData} 
          coachAvatar={profile?.data?.avatar_url || '/placeholder.png'}
          coachName={profile?.data?.first_name + ' ' + profile?.data?.last_name}
        />
      )
    },
  ];

  isMobile && tabs.push({
    label: 'Schedule',
    content: (
      <Row className={styles.scheduleRow}>
        <ProfileSchedule 
          schedule={gym?.data?.schedule as GymSchedule}
          days={days}
          selectedDay={getCurrentDay(daySelected)}
          daily
        />  
      </Row>
    )
  });

  const determineLayout = () => {
    if (isMobile) {
      return (
        <>
          <Row className={styles.datePickerRow}>
            <DatePicker 
              onDaySelect={(day) => {
                updateGymAnnouncements.mutate({ date: day });
                updateTechniques.mutate({ date: day });
                setDaySelected(day);
              }}
            />
          </Row>
          <Row className={styles.leftColumn}>
            <DashboardTabs 
              tabs={tabs}
              overrideTab={overrideTab}
              setOverrideTab={setOverrideTab}
            />
          </Row>
        </>
      )
    } else {
      return (
        <>
          <Col xs={8} className={styles.leftColumn}>
            <DashboardTabs 
              tabs={tabs}
              overrideTab={overrideTab}
              setOverrideTab={setOverrideTab}
            />
          </Col>
          <Col xs={4} className={styles.rightColumn}>
            <Row className={styles.datePickerRow}>
              <DatePicker 
                onDaySelect={(day) => {
                  updateGymAnnouncements.mutate({ date: day });
                  updateTechniques.mutate({ date: day });
                  setDaySelected(day);
                }}
              />
            </Row>
            <Row className={styles.scheduleRow}>
              <ProfileSchedule 
                schedule={gym?.data?.schedule as GymSchedule}
                days={days}
                selectedDay={getCurrentDay(daySelected)}
                daily
              />
            </Row>
            <Row className={styles.upgradeRow}>
              <Upgrade />
            </Row>
          </Col>
        </>
      )
    }
  }

  return (
    <>
      <Row className={styles.container}>
        {determineLayout()}
      </Row>
    </>
  );
};

export default MyGymPage;