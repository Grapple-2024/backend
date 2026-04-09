import React, { useState } from 'react';
import ReactPlayer from 'react-player';
import './styles.css'; // Your custom CSS

interface VideoPlayerProps {
  previewUrl: string;
}

const VideoPlayer: React.FC<VideoPlayerProps> = ({ previewUrl }) => {
  const [isPlaying, setIsPlaying] = useState(false);

  const handlePlay = () => {
    setIsPlaying(true);
  };

  const handlePause = () => {
    setIsPlaying(false);
  };

  return (
    <div className="video-container">
      {/* Custom play button overlay */}
      {!isPlaying && (
        <div className="play-button" onClick={handlePlay}>
          &#9658; {/* Unicode character for play button */}
        </div>
      )}
      <ReactPlayer
        url={previewUrl}
        playing={isPlaying}
        controls
        width="100%"
        height="100%"
        onPlay={handlePlay}
        onPause={handlePause}
      />
    </div>
  );
};

export default VideoPlayer;
