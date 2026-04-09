"use client";

import styles from './page.module.css';

import { useQuery } from "@tanstack/react-query";
import { useParams, useRouter } from "next/navigation";
import PageSkeleton from "@/components/PageSkeleton";
import { Col, Row } from 'react-bootstrap';
import ImageSection from '@/components/Profile/ImageSection';
import LogoSection from '@/components/Profile/LogoSection';
import DescriptionSection from '@/components/Profile/DescriptionSection';
import HeroSection from '@/components/Profile/HeroSection';
import DisciplinesSection from '@/components/Profile/DisciplinesSection';
import DatePicker from '@/components/DatePicker';
import ProfileSchedule from '@/components/Profile/Display/Schedule';
import { useState } from 'react';
import { useCreateRequest, useGetByRequestor } from '@/hook/request';
import { useGetUserProfile } from '@/hook/profile';
import { getGymById } from '@/hook/gym';

const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

const GymPage = () => {
  const { gymId } = useParams();
  const router = useRouter();
  
  const gymData = useQuery({
    queryKey: ['gyms', gymId],
    queryFn: () => getGymById(gymId as string),
    retry(failureCount, error) {
      return failureCount < 0;
    },
    enabled: !!gymId,
  });
  const isCoach = false;

  const profile = useGetUserProfile();
  const userRequests = useGetByRequestor(gymId as string);
  const requestToJoin = useCreateRequest();
  const [daySelected, setDaySelected] = useState<Date>(new Date());
  
  const gym = gymData?.data;
  
  if (gymData.isPending) {
    return <PageSkeleton />
  }
  
  function getCurrentDay(daySelected: Date): string {
    const daysOfWeek = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];
    const currentDayIndex = daySelected.getDay();
    return daysOfWeek[currentDayIndex - 1 === -1 ? 6 : currentDayIndex - 1];
  }

  const getByType = (type: string, data: any[]) => {
    if (data) {
      for (const value of data) {
        if (value.status === type) {
          return true;
        }
      }
    }

    return false;
  };
  
  const getRequestButton = () => {
    if (isCoach || (!profile?.data)) {
      return (
        <button className={styles.primaryButton} onClick={() => {
          router.push('/auth');
        }}>
          Create an account to join
        </button>
      )
    }
    
    if (userRequests?.data?.length === 0) {
      return (
        <button className={styles.primaryButton} onClick={() => {
          requestToJoin.mutate({
            requestor_id: profile?.data?.cognito_id,
            gym_id: gymId,
            status: 'Pending',
            first_name: profile?.data?.first_name,
            last_name: profile?.data?.last_name,
            requestor_email: profile?.data?.email,
          });
        }}>
          Request to Join
        </button>
      );
    }
    const request = userRequests?.data;
    console.log("REQUEST: ", request);
    if (getByType('Accepted', request)) {
      return (
        <button className={styles.primaryButton} disabled>
          You are already a member of a gym
        </button>
      );
    }

    if (getByType('Pending', request)) {
      return (
        <button className={styles.primaryButton} disabled>
          Your request is pending
        </button>
      );
    }

    if (getByType('Denied', request)) {
      return (
        <button className={styles.primaryButton} disabled>
          Your request has been denied, please contact your coach
        </button>
      );
    }
  }
  
  return (
    <>
      <div className={styles.container}>
        <Row style={{ margin: 0, padding: 0 }}>
          <Col xs={12} md={8} style={{ backgroundColor: 'white', padding: 0 }}>
            <ImageSection 
              bannerUrl={gym?.banner_url || '/placeholder-banner.png'}
            />

            <div className={styles.contentGrid}>
              <div className={styles.mainContent}>
                <div className={styles.gymInfo}>
                  <LogoSection 
                    logoUrl={gym?.logo_url || '/placeholder-logo.jpeg'}
                    gym={gym}
                  />
                  <div className={styles.buttonGroup}>
                    {getRequestButton()}
                    {/* <button className={styles.secondaryButton} disabled>
                      Request Info
                    </button> */}
                  </div>
                </div>

                <section className={styles.section}>
                  <DescriptionSection 
                    gym={gym}
                  />
                </section>

                <section className={styles.videoSection}>
                  <HeroSection 
                    gym={gym}
                  />
                </section>

                <section className={styles.section}>
                  <DisciplinesSection 
                    gym={gym}
                  />
                </section>
              </div>
            </div>
          </Col>
          
          <Col xs={12} md={4} style={{ margin: 0, height: '92vh' }}>
            <Row style={{ margin: 0, paddingRight: 10, paddingTop: 10, height: '12vh' }}>
              <DatePicker 
                onDaySelect={(day) => {
                  setDaySelected(day);
                }}
                isProfilePage
              />
            </Row>
            <Row style={{ margin: 0, paddingRight: 10, paddingTop: 30, height: '70vh' }}>
              <ProfileSchedule 
                schedule={gym?.schedule}
                days={days}
                selectedDay={getCurrentDay(daySelected)}
                daily
              />
            </Row>
          </Col>
        </Row>
      </div>
    </>
  );
};

export default GymPage;
