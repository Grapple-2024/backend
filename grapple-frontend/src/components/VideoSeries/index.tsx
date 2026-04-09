import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import styles from './VideoSeries.module.css';
import DisciplineTags from '@/components/DisciplineTags';
import { Image } from 'react-bootstrap';

interface VideoSeriesProps {
  series: any[];
  setVideos: (videos: any[]) => void;
  title?: string
}

const VideoSeries: React.FC<VideoSeriesProps> = ({ series, title }) => {
  const [thumbnails, setThumbnails] = useState<{ [key: string]: string }>({});
  const router = useRouter(); // Initialize useRouter

  const generateThumbnail = (videoUrl: string, seriesId: string) => {
    const videoElement = document.createElement('video');
    videoElement.src = videoUrl;
    videoElement.crossOrigin = 'anonymous';
    videoElement.currentTime = 2; // Capture frame at 2 seconds
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
    series?.forEach((seriesItem) => {
      if (seriesItem?.videos?.length > 0 && !thumbnails[seriesItem.pk]) {
        generateThumbnail(seriesItem.videos[0].presigned_url, seriesItem.pk);
      }
    });
  }, [series, thumbnails]);

  const handleCardClick = (seriesId: string) => {
    router.push(`/student/content/${seriesId}`); // Navigate to the desired route
  };

  return (
    <div className={styles.seriesWrapperContainer}>
      <h3 className={styles.title}>{title ? title : "This Week's Techniques"}</h3>
      <div className={styles.seriesList}>
        {series?.length === 0 ? (
          <div className={styles.noSeries}>
            <p>No series available yet. Please check back later.</p>
          </div>
        ) : (
          series && series?.map((seriesItem) => (
            <>
              {seriesItem?.videos?.length > 0 && (
                <div
                  key={seriesItem.pk}
                  className={styles.seriesWrapper}
                  onClick={() => handleCardClick(seriesItem.pk)} // Add click handler
                >
                  <div className={styles.thumbnailContainer}>
                    <span className={styles.difficulty}>{seriesItem.difficulties[0]}</span>
                    <Image
                      src={thumbnails[seriesItem.pk] || 'placeholder.png'}
                      alt={`Thumbnail for ${seriesItem.title}`}
                      className={styles.thumbnail}
                    />
                    <div className={styles.videoDuration}>
                      {seriesItem?.videos?.length}
                      {seriesItem?.videos?.length > 0 ? ' videos' : ' video'}
                    </div>
                  </div>
                  <div className={styles.textContainer}>
                    <h4 className={styles.seriesTitle}>{seriesItem.title}</h4>
                    <DisciplineTags disciplinesValues={seriesItem.disciplines} />
                    <div className={styles.infoRow}>
                      <Image
                        src={seriesItem.coachAvatar || '/placeholder.png'}
                        alt="Coach Avatar"
                        className={styles.coachAvatar}
                      />
                      <span className={styles.coachName}>Adam Watts</span>
                    </div>
                  </div>
                </div>
              )}
            </>
          ))
        )}
      </div>
    </div>
  );
};

export default VideoSeries;
