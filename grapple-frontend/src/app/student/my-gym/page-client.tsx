"use client";

import StudentDashboard from "./components/StudentDashboard";

const StudentMyGymPage = ({
  announcements,
  isAcceptedUser,
}: any) => {
  return (
    <>
      <StudentDashboard intialAnnouncements={announcements} isAcceptedUser={isAcceptedUser} />
    </>
  );
};

export default StudentMyGymPage;