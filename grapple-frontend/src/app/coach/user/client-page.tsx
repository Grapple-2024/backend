// components/UsersPageClient.tsx
'use client';

import React, { useState } from 'react';
import styles from './UsersPage.module.css';
import UsersTable from '@/components/UserTable';
import UserHeader from '@/components/UserHeader';
import UploadEmailModal from '@/components/UploadEmailModal';

interface UsersPageClientProps {
  requests: any[];
  role: any;
  gym: any;
}

const filters = [
  {
    label: "In-person",
    value: "IN-PERSON",
    field: "membership_type",
  },
  {
    label: "Virtual",
    value: "VIRTUAL",
    field: "membership_type",
  },
  {
    label: "Coach",
    value: "coach",
    field: "role",
  },
  {
    label: "Owner",
    value: "owner",
    field: "role",
  },
  {
    label: "Student",
    value: "student",
    field: "role",
  },
  {
    label: "Accepted",
    value: "Accepted",
    field: "status",
  },
  {
    label: "Denied",
    value: "Denied",
    field: "status",
  },
  {
    label: "Pending",
    value: "Pending",
    field: "status",
  },
];

const UsersPageClient: React.FC<UsersPageClientProps> = ({ requests, role, gym }) => {
  const [selectedRows, setSelectedRows] = useState<number[]>([]);
  const [show, setShow] = useState(false);
  
  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <UserHeader totalMembers={requests?.length} gymId={gym?.id}/>
      </div>
      <div className={styles.table}>
        <UsersTable 
          role={role}
          initialData={requests} 
          gym={gym}
          setShow={() => setShow(true)}
          filters={filters}
        />
      </div>
      <UploadEmailModal 
        show={show} 
        onHide={() => setShow(false)} 
        onSubmit={async () => console.log("HI")} 
      />
    </div>
  );
};

export default UsersPageClient;