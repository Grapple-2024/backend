"use client";

import React, { useEffect, useState } from "react";
import SeriesSidebar from "@/components/SeriesSidebar";
import { Col, Row } from "react-bootstrap";
import { useParams } from "next/navigation";
import PageSkeleton from "@/components/PageSkeleton";
import VideoPlayerContent from "@/components/VideoPlayerContent";
import { useEditSeriesContext } from "@/context/edit-series";
import { useGetSeriesView } from "@/hook/series";

const DesktopView: React.FC<any> = () => {
  const { id } = useParams();
  const series: any = useGetSeriesView(id as string);
  const { 
    setCurrentSeries, 
    currentSelection, 
    setCurrentSelection 
  } = useEditSeriesContext();
  const [selectedVideo, setSelectedVideo] = useState<any>(null);
  
  useEffect(() => {
    if (!series.isPending && series?.data?.videos?.length > 0) {
      setCurrentSelection(series?.data?.videos[0]);
      setCurrentSeries(series?.data);
    }
  }, [series.isPending, series.data]);

  useEffect(() => {
    const newSelectedVideo = series?.data?.videos?.find((video: any) => video?.id === currentSelection?.id);
    
    setSelectedVideo(newSelectedVideo);
  }, [currentSelection]);
  
  if (series.isPending && !id) {
    return <PageSkeleton />;
  }
  
  return (  
    <>
      <Row style={{ margin: 0, padding: 0, height: '100%', minHeight: 0 }}>
        <Col xs={9} style={{ 
          margin: 0, 
          padding: '0px 30px 0px 30px', 
          overflow: 'auto', 
          height: '100%' 
        }}>
          <VideoPlayerContent 
            seriesTitle={series.data?.title}
            seriesDescription={series.data?.description}
            videoSrc={selectedVideo?.presigned_url}
            videoTitle={selectedVideo?.title}
            disciplines={series.data?.disciplines}
            coachAvatar={series.data?.coach_avatar}
            coachName={series.data?.coach_name}
            videoDescription={selectedVideo?.description}
            isCoach
          />
        </Col>
        <Col xs={3} style={{ margin: 0, padding: 0 }}>
          <div style={{ height: '100%', padding: 0, marginRight: 0, minHeight: 0 }}>
            <SeriesSidebar 
              coachName={series.data?.coach_name}
              coachAvatar={series.data?.coach_avatar}
              videos={series.data?.videos as any} 
              currentSelection={currentSelection} 
              setCurrentSelection={setCurrentSelection}
              isCoach
            />
          </div>
        </Col>
      </Row>
    </>
  );
};

export default DesktopView;
