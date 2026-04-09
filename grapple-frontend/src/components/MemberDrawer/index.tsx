'use client';

import { useState } from 'react';
import { Offcanvas, Nav } from 'react-bootstrap';
import { User } from '@/components/UserTable';
import OverviewTab from './tabs/OverviewTab';
import BillingTab from './tabs/BillingTab';
import AttendanceTab from './tabs/AttendanceTab';
import BeltTab from './tabs/BeltTab';
import NotesTab from './tabs/NotesTab';
import styles from './MemberDrawer.module.css';

export interface MemberDrawerProps {
  member: User | null;
  onClose: () => void;
}

type Tab = 'overview' | 'billing' | 'attendance' | 'belt' | 'notes';

const TABS: { key: Tab; label: string }[] = [
  { key: 'overview',    label: 'Overview' },
  { key: 'billing',     label: 'Billing' },
  { key: 'attendance',  label: 'Attendance' },
  { key: 'belt',        label: 'Belt History' },
  { key: 'notes',       label: 'Notes' },
];

export default function MemberDrawer({ member, onClose }: MemberDrawerProps) {
  const [activeTab, setActiveTab] = useState<Tab>('overview');

  const handleClose = () => {
    setActiveTab('overview');
    onClose();
  };

  const fullName = member ? `${member.first_name} ${member.last_name}` : '';

  return (
    <Offcanvas
      show={!!member}
      onHide={handleClose}
      placement="end"
      className={styles.drawer}
    >
      <Offcanvas.Header className={styles.drawerHeader}>
        <div className={styles.drawerHeaderInner}>
          {member?.profile?.avatar_url ? (
            <img
              src={member.profile.avatar_url}
              alt={fullName}
              className={styles.drawerAvatar}
            />
          ) : (
            <div className={styles.drawerAvatarPlaceholder}>
              {member?.first_name?.[0]}{member?.last_name?.[0]}
            </div>
          )}
          <div className={styles.drawerMemberInfo}>
            <span className={styles.drawerName}>{fullName}</span>
            <span className={styles.drawerRole}>{member?.role || 'Member'}</span>
          </div>
          <button className={styles.closeBtn} onClick={handleClose} aria-label="Close">
            ×
          </button>
        </div>

        <Nav className={styles.tabNav} activeKey={activeTab}>
          {TABS.map(t => (
            <Nav.Item key={t.key}>
              <Nav.Link
                eventKey={t.key}
                className={`${styles.tabLink} ${activeTab === t.key ? styles.tabLinkActive : ''}`}
                onClick={() => setActiveTab(t.key)}
              >
                {t.label}
              </Nav.Link>
            </Nav.Item>
          ))}
        </Nav>
      </Offcanvas.Header>

      <Offcanvas.Body className={styles.drawerBody}>
        {member && (
          <>
            {activeTab === 'overview'   && <OverviewTab   member={member} />}
            {activeTab === 'billing'    && <BillingTab    member={member} />}
            {activeTab === 'attendance' && <AttendanceTab member={member} />}
            {activeTab === 'belt'       && <BeltTab member={member} />}
            {activeTab === 'notes'      && <NotesTab />}
          </>
        )}
      </Offcanvas.Body>
    </Offcanvas>
  );
}
