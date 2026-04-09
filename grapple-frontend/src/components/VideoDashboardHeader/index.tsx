import React from 'react';
import { Button, Col, Image, Row } from "react-bootstrap";
import styles from './VideoDashboardHeader.module.css';
import VideoFilterButton from '../VideoFilterButton';
import { useContentContext } from '@/context/content';
import { useEditSeriesContext } from '@/context/edit-series';

interface VideoDashboardHeaderProps {
  create?: boolean;
  isCoach?: boolean;
}

const VideoDashboardHeader = ({
  create = true,
  isCoach = false,
}: VideoDashboardHeaderProps) => {
  const { setOpen } = useContentContext();
  const { setIsEditing, setStep } = useEditSeriesContext();
  
  return (
    <Row>
      <Col md={8} xs={isCoach ? 12 : 8} className={styles.colContainer}>
        <h4 className={styles.title}>
          Discover a Series
        </h4>
      </Col>
      <Col md={4} xs={isCoach ? 12 : 4} className={styles.buttonCol}>
        <VideoFilterButton isCoach={isCoach}/>
        {
          create && (
            <Button 
              variant='dark' 
              className={styles.buttonStyle}
              onClick={() => {
                setIsEditing(false);
                setStep(1);
                setOpen(true);
              }}
            >
              <Image 
                src={'/create-series-button.svg'}
                className={styles.imageIcon}
                alt="Create Series"
              />
              <span className={styles.buttonText}>Create</span>
            </Button>
          )
        }
      </Col>
    </Row>
  );
};

export default VideoDashboardHeader;
