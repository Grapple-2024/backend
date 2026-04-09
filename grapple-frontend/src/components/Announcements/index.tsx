import React, { useState } from 'react';
import { Row, Container, Form, Button } from 'react-bootstrap';
import { FaTrash } from 'react-icons/fa';
import Avatar from '@/components/Avatar';
import EditableCard from '../EditableCard';
import styles from './Announcements.module.css';

interface AnnouncementsData {
  created_at: string;
  title: string;
  content: string;
  id: string;
  coach_name: string;
  coach_avatar: string;
}

interface AnnouncementsProps {
  create?: boolean;
  onShow?: () => void;
  data: AnnouncementsData[];
  onDelete?: (id: string) => void;
  onSave?: (newAnnouncement: any) => void;
  coachName: string;
  coachAvatar: string;
  quickAction?: boolean;
  isCoach?: boolean;
}

const AnnouncementEditCard = ({
  onCancel,
  onSave,
  coachName,
  coachAvatar,
  newTitle,
  setNewTitle,
  newContent,
  setNewContent,
  currentDate,
}: any) => (
  <div className={styles.announcementCard}>
    <div className={styles.cardHeader}>
      <Avatar src={coachAvatar} height={50} />
      <div className={styles.cardTitle}>
        <p className={styles.coachName}>{coachName}</p>
        <div>
          <Form.Control
            type="text"
            placeholder="Enter title"
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            className={styles.announcementTitle}
          />
          {" -"}
          <span className={styles.announcementDate}>{currentDate}</span>
        </div>
      </div>
    </div>
    <Container className={styles.cardContent}>
      <Form.Control
        as="textarea"
        placeholder="Enter announcement content"
        rows={3}
        value={newContent}
        onChange={(e) => setNewContent(e.target.value)}
      />
    </Container>
    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
      <Button
        variant="secondary"
        className={styles.cancelButton}
        onClick={onCancel}
      >
        Cancel
      </Button>
      <Button
        variant="primary"
        className={styles.saveButton}
        onClick={() => {
          onSave();
          onCancel();
        }}
      >
        Post
      </Button>
    </div>
  </div>
);

const Announcements = ({
  onShow,
  data,
  onDelete,
  onSave,
  coachName,
  coachAvatar,
  quickAction = false,
  create = false,
  isCoach=false
}: AnnouncementsProps) => {
  const [newTitle, setNewTitle] = useState('');
  const [newContent, setNewContent] = useState('');

  const handleSave = () => {
    const newAnnouncement = {
      title: newTitle,
      content: newContent,
      coach_name: coachName,
      coach_avatar: coachAvatar,
    };
    onSave && onSave(newAnnouncement);
    setNewTitle('');
    setNewContent('');
  };

  const currentDate = new Date().toLocaleDateString();

  return (
    <Container className={styles.announcementContainer}>
      {
        create && (
          <Row className={styles.createRow}>
            <EditableCard
              createText="Create Announcement"
              quickAction={quickAction}
              editComponent={
                <AnnouncementEditCard
                  onSave={handleSave}
                  coachName={coachName}
                  coachAvatar={coachAvatar}
                  newTitle={newTitle}
                  setNewTitle={setNewTitle}
                  newContent={newContent}
                  setNewContent={setNewContent}
                  currentDate={currentDate}
                />
              }
            />
          </Row>
        )
      }

      <div className={styles.announcementList}>
        {data?.length > 0 ? data.map((announcement, index) => {
          const date = new Date(announcement.created_at);
          const cleanDate = date.toLocaleDateString();

          return (
            <div className={styles.announcementCard} key={index}>
              <div className={styles.cardHeader}>
                <Avatar src={announcement?.coach_avatar} height={60} />
                <div className={styles.cardTitle}>
                  <p className={styles.coachName}>{announcement?.coach_name}</p>
                  <h5 className={styles.announcementTitle}>
                    {announcement.title} -<span className={styles.announcementDate}>{" " + cleanDate}</span>
                  </h5>
                </div>
              </div>
              {isCoach && (
                <FaTrash
                  size={20}
                  className={styles.trashIcon}
                  onClick={() => {
                    onDelete && onDelete(announcement.id);
                  }}
                />
              )}
              <Container className={styles.cardContent}>
                <p>{announcement.content}</p>
              </Container>
            </div>
          );
        }) : (
          <div className={styles.noAnnouncements}>
            <h3>No Announcements</h3>
          </div>
        )}
      </div>
    </Container>
  );
};

export default Announcements;
