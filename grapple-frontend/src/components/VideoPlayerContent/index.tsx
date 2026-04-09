import { Col, Dropdown, Image, Row } from "react-bootstrap";
import VideoSeriesHeader from "../VideoSeriesHeader";
import VideoPlayer from "@/app/components/VideoPlayer";
import DisciplineTags from "../DisciplineTags";
import ExpandableText from "../ExpandableText";
import { CiBookmark } from "react-icons/ci";
import { useState } from "react";
import { useEditSeriesContext } from "@/context/edit-series";
import styles from './VideoPlayerContent.module.css';

interface Props {
  seriesTitle: string;
  videoSrc: string;
  videoTitle: string;
  disciplines: string[];
  coachAvatar: string;
  coachName: string;
  videoDescription: string;
  isCoach?: boolean;
  seriesDescription?: string;
}

const VideoPlayerContent = ({
  seriesTitle,
  seriesDescription = '',
  videoSrc,
  videoTitle,
  disciplines,
  coachAvatar,
  coachName,
  videoDescription,
  isCoach = false
}: Props) => {

  return (
    <Row className={styles.container}>
      <VideoSeriesHeader 
        title={seriesTitle} 
        seriesDescription={seriesDescription} 
        isCoach={isCoach} 
      />
      
      <Row className={styles.videoContainer}>
        <VideoPlayer key={videoSrc} previewUrl={videoSrc} />
      </Row>
      
      <Row className={styles.spacer}></Row>
      
      <Row className={styles.titleRow}>
        <Col xs={6} className={styles.titleCol}>
          <h4 className={styles.videoTitle}>{videoTitle}</h4>
        </Col>
        <Col className={styles.disciplineCol}>
          <DisciplineTags disciplinesValues={disciplines} />
        </Col>
      </Row>
      
      <Row className={styles.spacer}></Row>
      
      <Row className={styles.coachInfo}>
        <Col xs={1}>
          <Image 
            src={coachAvatar || '/placeholder.png'} 
            alt="Coach Avatar" 
            className={styles.coachAvatar}
          />
        </Col>
        <Col xs={2} style={{ display: 'flex', justifyContent: 'flex-start' }}>
          <span className={styles.coachName}>
            {coachName}
          </span>
        </Col>
        <Col xs={9} className={styles.buttonContainer}>
          <div className={styles.actionButton}>
            <CiBookmark className={styles.actionIcon} />
            <h6 className={styles.actionText}>
              Bookmark
            </h6>
          </div>
          <div className={styles.actionButtonShare}>
            <Image 
              src={'/share-button.svg'}
              className={styles.actionIcon}
              alt="Share Button" 
            />
            <h6 className={styles.actionText}>
              Share
            </h6>
          </div>
        </Col>
      </Row>
      
      <Row className={styles.spacer}></Row>
      
      <Row className={styles.descriptionContainer}>
        <Row className={styles.descriptionContent}>
          <ExpandableText 
            text={videoDescription} 
            maxLength={35} 
          />
        </Row>
      </Row>
    </Row>
  );
};

export default VideoPlayerContent;