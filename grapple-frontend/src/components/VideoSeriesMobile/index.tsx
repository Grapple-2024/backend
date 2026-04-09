import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation'; 
import styles from './VideoSeriesMobile.module.css';
import DisciplineTags from '@/components/DisciplineTags';
import { Image } from 'react-bootstrap';

interface VideoSeriesMobileProps {
  series: any[];
  setVideos: (videos: any[]) => void;
  title?: string
}

const VideoSeriesMobile: React.FC<VideoSeriesMobileProps> = ({ series, title }) => {
  const [thumbnails, setThumbnails] = useState<{ [key: string]: string }>({});
  const router = useRouter(); 

  const generateThumbnail = (videoUrl: string, seriesId: string) => {
    const videoElement = document.createElement('video');
    videoElement.src = videoUrl;
    videoElement.crossOrigin = 'anonymous';
    videoElement.currentTime = 2; 
    videoElement.muted = true;

    videoElement.addEventListener('canplay', () => {
      const canvas = document.createElement('canvas');
      canvas.width = 160;
      canvas.height = 90;
      const ctx = canvas.getContext('2d');
      ctx?.drawImage(videoElement, 0, 0, canvas.width, canvas.height);
      const thumbnailUrl = canvas.toDataURL('image/png');

      setThumbnails((prevThumbnails) => ({
        ...prevThumbnails,
        [seriesId]: thumbnailUrl,
      }));

      videoElement.remove();
    });
  };

  useEffect(() => {
    series.forEach((seriesItem) => {
      if (seriesItem?.videos?.length > 0 && !thumbnails[seriesItem.id]) {
        generateThumbnail(seriesItem.videos[0].presigned_url, seriesItem.id);
      }
    });
  }, [series, thumbnails]);

  const handleCardClick = (seriesId: string) => {
    router.push(`/student/content/${seriesId}`); 
  };

  return (
    <div className={styles.seriesWrapperContainer}>
      <h3 className={styles.title}>{title ? title : "This Week's Techniques"}</h3>
      <div className={styles.seriesList}>
        {series.length === 0 ? (
          <div className={styles.noSeries}>
            <p>No series available yet. Please check back later.</p>
          </div>
        ) : (
          series.map((seriesItem) => (
            <div
              key={seriesItem.id}
              className={styles.seriesWrapper}
              onClick={() => handleCardClick(seriesItem.id)} 
            >
              <div className={styles.thumbnailContainer}>
                <Image
                  src={thumbnails[seriesItem.id] || 'placeholder.png'}
                  alt={`Thumbnail for ${seriesItem.title}`}
                  className={styles.thumbnail}
                />
                {/* <span className={styles.difficulty}>{seriesItem.difficulties[0]}</span> */}
                <div className={styles.videoDuration}>
                  {seriesItem?.videos?.length}
                  {seriesItem?.videos?.length > 1 ? ' videos' : ' video'}
                </div>
              </div>
              <div className={styles.videoInfo}>
                <h4 className={styles.seriesTitle}>{seriesItem.title}</h4>
                <DisciplineTags disciplinesValues={seriesItem.disciplines} />
                <div className={styles.coachInfo}>
                  <Image
                    src={seriesItem.coachAvatar || '/placeholder.png'}
                    alt="Coach Avatar"
                    className={styles.coachAvatar}
                  />
                  <span className={styles.coachName}>Adam Watts</span>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default VideoSeriesMobile;
