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
const series = useGetSeriesView(id as string);
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
  const newSelectedVideo = series?.data?.videos?.find((video: any) => video.id === currentSelection.id);
  
  setSelectedVideo(newSelectedVideo);
}, [currentSelection]);

  if (series.isPending) {
    return <PageSkeleton />;
  }
  
  return (  
    <>
      <div style={{ marginLeft: 0, height: '92vh' }}>
        <Row style={{ margin: 0, padding: 0 }}>
          <Col xs={9} style={{ margin: 0, paddingRight: 30, padding: 10 }}>
            <VideoPlayerContent 
              seriesTitle={series.data?.title}
              videoSrc={selectedVideo?.presigned_url}
              videoTitle={selectedVideo?.title}
              disciplines={series.data?.disciplines}
              coachAvatar={series.data?.coach_avatar}
              coachName={series.data?.coach_name}
              videoDescription={selectedVideo?.description}
            />
          </Col>
          <Col xs={3} style={{ margin: 0, padding: 0 }}>
            <div style={{ height: '92vh', padding: 0, marginRight: 0 }}>
              <SeriesSidebar 
                coachName={series.data?.coach_name}
                coachAvatar={series.data?.coach_avatar}
                videos={series.data?.videos as any} 
                currentSelection={currentSelection} 
                setCurrentSelection={setCurrentSelection}
              />
            </div>
          </Col>
        </Row>
      </div>
    </>
  );
};

export default DesktopView;
