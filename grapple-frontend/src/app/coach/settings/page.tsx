'use client';

import { Row } from "react-bootstrap";
import ProfileCard from "../../../components/ProfileCard";
import BasicInfo from "../../../components/BasicInfo";
import ChangePassword from "../../../components/ChangePassword";
import DeleteAccount from "../../../components/DeleteAccount";
import { useQueryClient } from "@tanstack/react-query";

import PageSkeleton from "@/components/PageSkeleton";
import { useMessagingContext } from "@/context/message";
import Notifications from "@/components/Notifications";
import { useGetGym } from "@/hook/gym";
import { useGetUserProfile, useUpdateProfile } from "@/hook/profile";
  
const sidebarData = [
  { title: 'My Gym', route: '/my-gym', line: false },
  { title: 'Content', route: '/content', line: false },
  { title: 'Settings', route: `/settings`, line: true, active: true},
];

const SettingsPage = () => {
  const gymData = useGetGym();

  const profile = useGetUserProfile();
  const updateProfile = useUpdateProfile();

  if (gymData?.isPending) {
    return <PageSkeleton />
  }
  
  const data = {
    first_name: profile?.data?.first_name,
    last_name: profile?.data?.last_name,
    email: profile?.data?.email,
    phone_number: profile?.data?.phone_number,
    address: profile?.data?.address,
    city: profile?.data?.city,
    state: profile?.data?.state,
    zip: profile?.data?.zip,
  };
  
  return (
    <>
      <div style={{
        padding: 0,
        height: '100%',
        overflow: 'auto',
      }}>
        <Row id="profile-card" style={{ margin: 30 }}>
          <ProfileCard 
            src={profile?.data?.avatar_url ? profile?.data?.avatar_url : '/placeholder.png'}
            firstName={data ? data?.first_name + ' ' + data?.last_name: ''}
            title={'Coach'}
          />
        </Row>
        <Row style={{ margin: 30 }} id="basic-info">
          <BasicInfo user={data} onSave={async (values) => {
            updateProfile.mutate({
              ...profile?.data,
              ...values,
            });
          }}/>
        </Row>
        <Row style={{ margin: 30 }} id="change-password">
          <ChangePassword user={data} />
        </Row>
        <Row style={{ margin: 30 }}>
          <Notifications 
            value={profile?.data?.gyms?.length > 0 && profile?.data?.gyms[0]?.email_preferences?.notify_on_requests}
            onSave={(value: boolean) => {
              let updatedProfile = profile?.data;
              
              if (updatedProfile?.gyms?.length > 0) {
                updatedProfile.gyms[0].email_preferences.notify_on_requests = value;

                updateProfile.mutate(updatedProfile);
              }
            }}
          />
        </Row>
        <Row style={{ margin: 30 }} id="delete-account">
          <DeleteAccount />
        </Row>
      </div>
    </>
  );
};

export default SettingsPage;