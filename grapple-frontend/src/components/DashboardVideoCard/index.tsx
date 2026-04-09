import React, { useState } from 'react';
import { Card, Row, Col, Image, Dropdown } from 'react-bootstrap';
import { FaVideo, FaBookmark } from 'react-icons/fa';
import { MdVerified } from 'react-icons/md';
import styles from './DashboardVideoCard.module.css';
import DisciplineTags from '../DisciplineTags';
import { useRouter } from 'next/navigation';
import { HiOutlineDotsHorizontal } from 'react-icons/hi';
import { useEditSeriesContext } from '@/context/edit-series';
import ConfirmationModal from '../ConfirmationModal';
import { useDeleteSeries } from '@/hook/series';

interface DashboardVideoCardProps {
  disciplines: string[];
  coachName: string;
  title: string;
  videoCount: number;
  thumbnail: string;
  seriesId: string;
  coachAvatar: string;
  route?: string;
  difficulty: string;
}


const CustomToggle = React.forwardRef<HTMLAnchorElement, { onClick: (e: React.MouseEvent<HTMLAnchorElement>) => void }>(
  ({ onClick }, ref) => (
    <a
      href=""
      ref={ref}
      onClick={(e) => {
        e.preventDefault();
        onClick(e);
      }}
      style={{ color: 'inherit', textDecoration: 'none' }}
    >
      <HiOutlineDotsHorizontal size={30} />
    </a>
  )
);
const DashboardVideoCard: React.FC<DashboardVideoCardProps> = ({
  disciplines,
  coachName,
  title,
  difficulty,
  videoCount,
  thumbnail,
  coachAvatar,
  seriesId,
  route = 'student'
}) => {
  const router = useRouter();
  const {
    setIsEditing,
    setStep,
  } = useEditSeriesContext();
  const [show, setShow] = useState(false);
  const deleteSeries = useDeleteSeries();

  const handleDelete = (id: string) => {
    deleteSeries.mutate(id);
  };

  const handleEdit = () => {
    setIsEditing(true);
    setStep(1);
    router.push(`/${route}/content/${seriesId}`);
  };
  
  return (
    <div style={{ height: '100%', backgroundColor: '#F1F5F9' }}>
      <div className={styles.topLine} />
      <div className={styles.bottomLine} />
      <Card 
        className={styles.videoCard} 
        style={{ 
          height: '100%', 
          display: 'flex', 
          flexDirection: 
          'column', 
          backgroundColor: '#F1F5F9' 
        }}
      >
        <div 
          className={styles.videoThumbnailContainer} 
          style={{ flexGrow: 1 }}
          onClick={() => {
            setIsEditing(false);
            setStep(1);
            router.push(`/${route}/content/${seriesId}`);
          }}
        >
          <Image
            src={thumbnail}
            alt="Video Thumbnail"
            className={styles.videoThumbnail}
          />
          <div className={styles.videoOverlayInfo}>
            <Row>
              <Col className="text-right">
                <FaVideo className={styles.videoIconSmall} /> {videoCount} Videos
              </Col>
            </Row>
          </div>
          <div className={styles.difficultyOverlay}>
            {difficulty}
          </div>
        </div>
        <Card.Body className={styles.videoDetails} style={{ flexShrink: 0 }}>
          <Row>
            <Card.Title className={styles.videoTitle}>
              {title}
              
              <div style={{
                  margin: 0,
                  display: 'flex',
                  justifyContent: 'flex-end',
                }}>
                  <Dropdown>
                    <Dropdown.Toggle as={CustomToggle}  />
                    <Dropdown.Menu>
                      <Dropdown.Item onClick={(e) => {
                        handleEdit();
                      }}>
                        Edit
                      </Dropdown.Item>
                      <Dropdown.Divider />
                      <Dropdown.Item onClick={(e) => {
                        setShow(true);
                      }}>
                        Delete
                      </Dropdown.Item>
                    </Dropdown.Menu>
                  </Dropdown>
                </div>
            </Card.Title>
          </Row>

          <Row className="align-items-center">
            <Col xs={2} className="text-center">
              <Image
                src={coachAvatar}
                alt="Coach Avatar"
                roundedCircle
                className={styles.coachAvatar}
              />
            </Col>
            <Col xs={10} className={styles.coachInfo}>
              <Card.Text className={styles.videoAuthor}>
                {coachName} <MdVerified className={styles.verifiedIcon} />
              </Card.Text>
            </Col>
          </Row>
          <Row style={{ marginTop: 10 }}>
            <Col xs={10} style={{ 
              display: 'flex', 
              justifyContent: "flex-start", 
              alignItems: 'center',
            }}>
              <DisciplineTags disciplinesValues={disciplines} />
            </Col>
            <Col style={{ 
              display: 'flex', 
              justifyContent: "flex-end", 
              alignItems: 'center'
            }}>
              <FaBookmark className={styles.iconBookmark} />
            </Col>
          </Row>
        </Card.Body>
      </Card>
      <ConfirmationModal 
        show={show}
        setShow={setShow}
        onConfirm={() => handleDelete(seriesId)}
      />
    </div>
  );
};

export default DashboardVideoCard;
