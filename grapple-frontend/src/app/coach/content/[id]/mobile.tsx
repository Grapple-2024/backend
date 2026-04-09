"use client";

import React, { useEffect, useMemo, useState } from "react";
import SeriesSidebar from "@/components/SeriesSidebar";
import VideoPlayer from "@/components/VideoPlayer";
import { Col, Image, Row } from "react-bootstrap";
import { useParams } from "next/navigation";
import PageSkeleton from "@/components/PageSkeleton";
import ExpandableText from "@/components/ExpandableText";
import DisciplineTags from "@/components/DisciplineTags";
import VideoSeriesHeader from "@/components/VideoSeriesHeader";
import { CiBookmark } from "react-icons/ci";
import { useEditSeriesContext } from "@/context/edit-series";
import styles from './mobile.module.css';
import { useGetSeriesView } from "@/hook/series";

const MobileView: React.FC<any> = () => {
  const { id } = useParams();
  const series = useGetSeriesView(id as string);
  const { 
    setCurrentSeries, 
    currentSelection, 
    setCurrentSelection 
  } = useEditSeriesContext();
  
  const [selectedVideo, setSelectedVideo] = useState<any>(null);
  
  // Initial setup of series and first video
  useEffect(() => {
    if (!series.isPending && series?.data?.videos?.length > 0) {
      setCurrentSelection(series?.data?.videos[0]);
      setCurrentSeries(series?.data);
    }
  }, [series.isPending, series.data]);
  
  // Use a more controlled effect for video selection
  useEffect(() => {
    if (series?.data?.videos) {
      const newSelectedVideo = series.data.videos.find((video: any) => video?.id === currentSelection?.id);
      if (newSelectedVideo) {
        setSelectedVideo(newSelectedVideo);
      }
    }
  }, [series?.data?.videos, currentSelection?.id]);
  
  const memoizedVideoPlayer = useMemo(() => (
    <VideoPlayer videoSrc={selectedVideo?.presigned_url} height={'29vh'}/>
  ), [selectedVideo?.presigned_url]);

  if (series.isPending) {
    return <PageSkeleton />;
  }
  
  return (  
    <>  
      <div className={styles.videoContainer}>
        <Row className={styles.rowContainer}>
          <VideoSeriesHeader title={series?.data?.title} seriesDescription={series?.data?.description} isCoach/>

          <Row className={styles.videoPlayerRow}>
            {memoizedVideoPlayer}
          </Row>
          <Row className={styles.videoInfoRow}>
            <Row className={styles.titleRow}>
              <Col className={styles.titleCol}>
                <h4>{selectedVideo?.title}</h4>
              </Col>
              <Col className={styles.tagsCol}>
                <DisciplineTags disciplinesValues={selectedVideo?.disciplines} />
              </Col>
            </Row>
            <Row className={styles.coachRow}>
              <Col xs={2} className={styles.avatarCol}>
                <Image 
                  src={series.data?.coach_avatar || '/placeholder.png'} 
                  alt="Coach Avatar" 
                  className={styles.coachAvatar}
                />
              </Col>
              <Col xs={10} className={styles.coachNameCol}>
                <span className={styles.coachName}>
                  {series.data?.coach_name}
                </span>
              </Col>
              <Col xs={12} className={styles.actionButtonsCol}>
                <div className={styles.actionButton}>
                  <CiBookmark className={styles.icon} />
                  <h6 className={styles.buttonText}>Bookmark</h6>
                </div>
                <div className={`${styles.actionButton} ${styles.shareButton}`}>
                  <Image 
                    src={'/share-button.svg'}
                    alt="Share Button"
                    className={styles.shareIcon}
                  />
                  <h6 className={styles.buttonText}>Share</h6>
                </div>
              </Col>
            </Row>
            <Row className={styles.descriptionRow}>
              <ExpandableText text={selectedVideo?.description} maxLength={35} />
            </Row>
          </Row>
          <Row className={styles.sidebarRow}>
            <SeriesSidebar 
              coachAvatar={series?.data?.coach_avatar || "/placeholder.png"}
              coachName={series?.data?.coach_name} 
              videos={series?.data?.videos as any} 
              currentSelection={currentSelection} 
              setCurrentSelection={setCurrentSelection}
              isCoach
            />
          </Row>
        </Row>
      </div>
    </>
  );
};

export default MobileView;
