// SeriesSidebar.tsx
import React, { forwardRef, useEffect, useState } from "react";
import styles from "./SeriesSidebar.module.css";
import { Button } from "react-bootstrap";
import { IoIosArrowUp, IoIosArrowDown } from "react-icons/io";
import { useContentContext } from "@/context/content";
import { FaPlus } from "react-icons/fa";
import { useEditSeriesContext } from "@/context/edit-series";
import SeriesSidebarCard from "../SeriesSidebarCard";
import { useUpdateSeries } from "@/hook/series";
import { useDeleteVideo } from "@/hook/video";

interface VideoItem {
  id: string;
  title: string;
  presigned_url: string;
  disciplines: string[];
  sort_order: number;
}

interface SeriesSidebarProps {
  videos: any[];
  coachName: string;
  coachAvatar: string;
  currentSelection: any;
  isCoach?: boolean;
  setCurrentSelection: (video: any) => void;
}

const SeriesSidebar: React.FC<SeriesSidebarProps> = ({ 
  videos: initialVideos, 
  currentSelection, 
  setCurrentSelection,
  coachAvatar,
  coachName, 
  isCoach = false
}) => {
  const [videos, setVideos] = useState<any[]>([]);
  const [durations, setDurations] = useState<string[]>([]);
  const [movingIndices, setMovingIndices] = useState<{from: number, to: number} | null>(null);
  const { setOpen } = useContentContext();
  const { 
    setIsEditing, 
    setStep, 
    setFormData, 
    setIsAdding,
    currentSeries, 
  } = useEditSeriesContext();
  const deleteVideo = useDeleteVideo();
  const updateSeries = useUpdateSeries(currentSeries?.id);
  
  useEffect(() => {
    if (initialVideos?.length > 0 ) {
      const sortedVideos = [...initialVideos].sort((a, b) => 
        (a.sort_order ?? 0) - (b.sort_order ?? 0)
      );
      setVideos(sortedVideos);
    }
  }, [initialVideos]);

  const handleEdit = (videoId: number) => {
    setCurrentSelection(videos[videoId]);
    setFormData({
      title: videos[videoId].title,
      description: videos[videoId]?.description,
      presigned_url: videos[videoId].presigned_url,
      thumbnail_url: videos[videoId].thumbnail_url,
      difficulty: videos[videoId]?.difficulty,
      disciplines: videos[videoId].disciplines,
      s3_object_key: videos[videoId].s3_object_key,
      sort_order: videos[videoId].sort_order,
      id: videos[videoId].id,
    });
    setIsEditing(true);
    setStep(2)
    setOpen(true);
  };

  const handleDelete = (videoId: string) => {
    deleteVideo.mutate({
      id: videoId,
      seriesId: currentSeries?.id,
    });
  };

  const handleMove = async (index: number, direction: 'up' | 'down') => {
    const newIndex = direction === 'up' ? index - 1 : index + 1;
    
    if (newIndex < 0 || newIndex >= videos.length) return;
    
    // Set moving indices for animation
    setMovingIndices({ from: index, to: newIndex });
    
    // Wait for animation to complete
    await new Promise(resolve => setTimeout(resolve, 300));
    
    const newVideos = [...videos];
    [newVideos[index], newVideos[newIndex]] = [newVideos[newIndex], newVideos[index]];
    
    // Update sort_order for all videos
    const updatedVideos = newVideos.map((video, idx) => ({
      ...video,
      sort_order: idx
    }));

    // Sort the videos by sort_order before setting state
    const sortedVideos = updatedVideos.sort((a, b) => a.sort_order - b.sort_order);
    setVideos(sortedVideos);

    // Reset moving indices
    setMovingIndices(null);

    updateSeries.mutate({
      ...currentSeries,
      videos: sortedVideos,
    });
  };

  useEffect(() => {
    const calculateDurationsAndThumbnails = async () => {
      const durationPromises = videos.map((video) => {
        return new Promise<string>((resolve) => {
          const videoElement = document.createElement("video");
          videoElement.src = video.presigned_url;

          videoElement.onloadedmetadata = () => {
            const duration = videoElement.duration;
            const minutes = Math.floor(duration / 60);
            const seconds = Math.floor(duration % 60);
            const formattedDuration = `${minutes}:${seconds < 10 ? "0" : ""}${seconds}`;
            resolve(formattedDuration);
          };

          videoElement.onerror = () => {
            resolve("Error");
          };
        });
      });

      const durations = await Promise.all(durationPromises);
      setDurations(durations);
    }

    calculateDurationsAndThumbnails();
  }, [videos]);
  
  return (
    <div className={styles.sidebar}>
      <div style={{ 
        display: "flex", 
        flexDirection: "row", 
        justifyContent: 'space-between',
        alignItems: 'center',
        marginTop: 10,
      }}>
        <h2 style={{ marginTop: 5 }}>Videos</h2>
        {isCoach && (
          <Button 
            className={styles.addVideoButton}
            onClick={() => {
              setIsEditing(false);
              setFormData({
                title: "",
                description: "",
                presigned_url: "",
                difficulty: "",
                disciplines: [],
                s3_object_key: "",
                sort_order: videos.length,
                id: "",
              });
              setStep(2);
              setIsAdding(true);
              setOpen(true);
            }} 
            style={{ margin: 0 }}
          >
            <FaPlus style={{ marginRight: 10 }}/>
            Video
          </Button>
        )}
      </div>
      {videos?.length > 0 && videos?.map((video, index) => {
        const isNext = index > 0 && videos[index - 1].id === currentSelection?.id;
        const isCurrent = video.id === currentSelection?.id;
        const isMoving = movingIndices?.from === index || movingIndices?.to === index;
        const moveDirection = movingIndices?.from === index ? 
          (movingIndices.to < index ? 'up' : 'down') : 
          (movingIndices?.to === index ? (movingIndices.from < index ? 'down' : 'up') : '');
        
        return (
          <div 
            key={video.id} 
            className={`${styles.videoContainer} ${isMoving ? styles.moving : ''}`}
          >
            {isNext && <div className={styles.upNext}>Up next</div>}
            {isCurrent && <div className={styles.upNext}>Now playing</div>}
            {isCoach && (
              <div className={styles.arrowControls}>
                <Button
                  variant="link"
                  onClick={(e) => {
                    e.stopPropagation();
                    if (!movingIndices) handleMove(index, 'up');
                  }}
                  disabled={index === 0 || movingIndices !== null}
                  style={{ 
                    padding: '0',
                    color: index === 0 ? '#ccc' : '#666',
                    cursor: index === 0 ? 'default' : 'pointer'
                  }}
                >
                  <IoIosArrowUp size={20} />
                </Button>
                <Button
                  variant="link"
                  onClick={(e) => {
                    e.stopPropagation();
                    if (!movingIndices) handleMove(index, 'down');
                  }}
                  disabled={index === videos.length - 1 || movingIndices !== null}
                  style={{ 
                    padding: '0',
                    color: index === videos.length - 1 ? '#ccc' : '#666',
                    cursor: index === videos.length - 1 ? 'default' : 'pointer'
                  }}
                >
                  <IoIosArrowDown size={20} />
                </Button>
              </div>
            )}
            <div className={`
              ${styles.videoItem} 
              ${currentSelection?.id === video.id ? styles.selected : ""} 
              ${moveDirection ? styles[`moving-${moveDirection}`] : ''}
            `}>
              <SeriesSidebarCard 
                currentSelection={currentSelection}
                setCurrentSelection={setCurrentSelection}
                video={video}
                durations={durations}
                index={index}
                handleEdit={handleEdit}
                handleDelete={handleDelete}
                coachAvatar={coachAvatar}
                coachName={coachName}
                isCoach={isCoach}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default SeriesSidebar;