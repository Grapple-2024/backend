import React, { useEffect, useState, useCallback } from 'react';
import styles from './TechniqueCard.module.css';
import { FaHeart, FaEye, FaArrowRight } from 'react-icons/fa';
import DisciplineTags from '@/components/DisciplineTags';
import ExpandableText from '../ExpandableText';
import { Popover, OverlayTrigger, Image } from 'react-bootstrap';
import { FaTrash } from "react-icons/fa";
import { CiBookmark } from "react-icons/ci";
import { useRouter } from 'next/navigation';

interface TechniqueCardProps {
  seriesItem: any;
  onDelete?: (id: string) => void;
  techniqueId?: string;
  path: string;
}

const TechniqueCard: React.FC<TechniqueCardProps> = ({ seriesItem, onDelete, techniqueId, path }) => {
  const videoViews = 0; // Hardcoded value for video views
  const router = useRouter();

  const popover = (
    <Popover id="popover-basic">
      <Popover.Body>
        Coming Soon!
      </Popover.Body>
    </Popover>
  );

  // Using useCallback to ensure the function is not re-created on each render
  const handleCardClick = useCallback(() => {
    router.push(`${path}/${seriesItem.id}`);
  }, [router, path, seriesItem.id]);
  return (
    <div className={styles.cardWrapper}>
      <div className={styles.thumbnailContainer}>
        <span className={styles.difficultyTag}>{seriesItem?.difficulties?.length > 0 && seriesItem.difficulties[0]}</span>
          <Image
            src={seriesItem?.videos?.length > 0 ? seriesItem?.videos[0]?.thumbnail_url : '/placeholder.png'} 
            className={styles.thumbnail}
          />
        <div className={styles.videoCountOverlay}>{`${seriesItem?.videos?.length} ${seriesItem?.videos.length > 1 ? 'videos' : 'video'}`}</div>
      </div>
      <div className={styles.textContainer}>
        <div>
          <div className={styles.headerRow}>
            <h4 className={styles.seriesTitle}>{seriesItem.title}</h4>
            {(onDelete && techniqueId) && (<FaTrash style={{ cursor: 'pointer' }} onClick={() => onDelete(techniqueId)}/>)}
          </div>
          <div className={styles.coachInfoRow}>
            <Image src={seriesItem?.coach_avatar || '/placeholder.png'} alt="Coach Avatar" className={styles.coachAvatar} />
            <span className={styles.coachName}>{seriesItem?.coach_name}</span>
          </div>
          <div className={styles.description}>
            <ExpandableText text={seriesItem.description} maxLength={50}/>
          </div>
          <DisciplineTags disciplinesValues={seriesItem?.disciplines} />
        </div>
        <div className={styles.footerRow}>
          <OverlayTrigger trigger={['hover', 'focus']} placement="top" overlay={popover}>
            <div className={styles.iconsRow}>
              <div className={styles.iconContainer}>
                <FaEye color='#CBD5E0' />
                <span className={styles.iconText}>{videoViews}</span>
                <CiBookmark className={styles.icon} />
              </div>
            </div>
          </OverlayTrigger>
          <button className={styles.arrowButton} onClick={handleCardClick}>
            <FaArrowRight />
          </button>
        </div>
      </div>
    </div>
  );
};

export default TechniqueCard;
