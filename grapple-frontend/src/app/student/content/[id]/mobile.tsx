"use client";

import React, { useEffect, useState } from "react";
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
      <div style={{ padding: 0, backgroundColor: 'white', margin: 0 }}>
        <Row style={{ margin: 0, padding: 0 }}>
          <VideoSeriesHeader title={series?.data?.title} />

          <Row style={{  
            margin: '10px 0',
            padding: 0,
            maxHeight: '60vh', // Restrict height to 60% of the viewport
            overflow: 'hidden', // Prevent any overflow
            aspectRatio: '16 / 9', 
          }}>
            <VideoPlayer key={selectedVideo?.presigned_url} videoSrc={selectedVideo?.presigned_url} height={'29vh'}/>
          </Row>
          <Row style={{ margin: 0, padding: '20px 20px 0px 20px' }}>
            <Row style={{ margin: 0, padding: 0 }}>
              <Col style={{ margin: 0, padding: 0 }}>
                <h4>{selectedVideo?.title}</h4>
              </Col>
              <Col style={{ display: 'flex', justifyContent: 'flex-end', margin: 0, padding: 0 }}>
                <DisciplineTags disciplinesValues={selectedVideo?.disciplines} />
              </Col>
            </Row>
            <Row style={{ display: 'flex', alignItems: 'center', margin: 0, padding: 0  }}>
              <Col xs={2} style={{ padding: 0}}>
                <Image 
                  src={series.data?.coach_avatar || '/placeholder.png'} 
                  alt="Coach Avatar" 
                  style={{ 
                    height: 50, 
                    width: 50, 
                    borderRadius: '50%', 
                    objectFit: 'cover', 
                  }} 
                />
              </Col>
              <Col xs={10} style={{ display: 'flex', justifyContent: 'flex-start' }}>
                <span style={{ fontSize: '14px', fontWeight: 'bold', marginLeft: 0, }}>
                  {series.data?.coach_name}
                </span>
              </Col>
              <Col xs={12} style={{ 
                display: 'flex', 
                justifyContent: 'center',
                marginTop: 10,
                marginBottom: 10, 
              }}>
                <div 
                  style={{ 
                    backgroundColor: '#F1F5F9', 
                    borderRadius: 18, 
                    padding: '10px 20px 10px 20px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    cursor: 'pointer'
                  }}
                >
                  <CiBookmark style={{ fontSize: 22, cursor: 'pointer' }} />
                  <h6
                    style={{ fontSize: 16, marginBottom: 0, marginLeft: 5, cursor: 'pointer' }}
                  >
                    Bookmark
                  </h6>
                </div>
                <div style={{ 
                  backgroundColor: '#F1F5F9', 
                  borderRadius: 18, 
                  padding: '10px 20px 10px 20px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  marginLeft: 20,
                  cursor: 'pointer'
                }}>
                  <Image 
                    src={'/share-button.svg'}
                    style={{
                      cursor: 'pointer',
                    }}
                    alt="Back Button" 
                  />
                  <h6 
                    style={{ fontSize: 16, marginBottom: 0, marginLeft: 5, cursor: 'pointer' }
                  }>
                    Share
                  </h6>
                </div>
              </Col>
            </Row>
            <Row style={{ margin: 0, padding: '20px 0px 20px 10px', marginTop: 10, backgroundColor: '#F1F4F9', borderRadius: 18 }}>
              <ExpandableText text={selectedVideo?.description} maxLength={35} />
            </Row>
          </Row>
          <Row style={{ margin: 0, padding: 0 }}>
            <SeriesSidebar 
              coachName={series.data?.coach_name}
              coachAvatar={series.data?.coach_avatar}
              videos={series.data?.videos as any} 
              currentSelection={currentSelection} 
              setCurrentSelection={setCurrentSelection}
            />
          </Row>
        </Row>
      </div>
    </>
  );
};

export default MobileView;
