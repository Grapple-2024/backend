import React, { useEffect, useRef } from 'react';
import ReactPlayer from 'react-player';
import styles from './VideoPlayer.module.css';

interface VideoPlayerProps {
  videoSrc: string;
  height: any;
}

const VideoPlayer: React.FC<VideoPlayerProps> = ({ videoSrc, height = '100%' }) => {
  const playerRef = useRef<ReactPlayer>(null);

  return (
    <div className={styles.videoContainer}>
      <div className={styles.videoWrapper}>
        <ReactPlayer
          ref={playerRef}
          url={videoSrc}
          width="100%"
          height={height}
          controls={true}
          playsinline={false}
          pip={false}
          config={{
            file: {
              attributes: {
                controlsList: 'nodownload',
                disablePictureInPicture: true,
              },
            },
          }}
        />
      </div>
    </div>
  );
};

export default VideoPlayer;