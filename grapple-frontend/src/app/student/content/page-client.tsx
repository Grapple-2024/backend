'use client';

import { Col, Placeholder, Row } from "react-bootstrap";
import { CSSTransition, TransitionGroup } from 'react-transition-group';
import './styles.css';
import { useMobileContext } from "@/context/mobile";
import DashboardVideoCard from "@/components/DashboardVideoCard";
import VideoDashboardHeader from "@/components/VideoDashboardHeader";
import SeriesPagination from "@/components/SeriesPagination";
import { useGetSeries } from "@/hook/series";
import { useGetStudentsRequest } from "@/hook/request";

const StudentContentPage = ({
  initialData
}: any) => {
  const { isMobile } = useMobileContext();
  
  const series = useGetSeries(initialData);
  const requests = useGetStudentsRequest();

  const status = requests?.data?.[0]?.status;
  const isUserAccepted = status === 'Accepted';

  const seriesComponent = () => {
    const chunkSeries = (data: any[], size: number) => {
      const chunks = [];
      for (let i = 0; i < data.length; i += size) {
        chunks.push(data.slice(i, i + size));
      }
      return chunks;
    };
  
    // Split series data into chunks of 3
    const seriesChunks = chunkSeries(series?.data?.data || [], 3);
    
    return (
      <>
        <TransitionGroup>
          {seriesChunks.map((chunk: any[], chunkIndex: number) => (
            <CSSTransition
              key={chunkIndex}
              timeout={500}
              classNames="series"
            >
              <Row key={chunkIndex} style={{ backgroundColor: '#F1F5F9 !important' }}>
                {chunk.map((series: any, index: number) => {
                  return ((
                    <Col
                      key={series.id}
                      style={{ height: '100%', backgroundColor: '#F1F5F9 !important', marginBottom: 20 }}
                      xs={isMobile ? 12 : 4} // Full width (12 columns) for mobile, 3 columns for desktop
                    >
                      <DashboardVideoCard
                        title={series.title}
                        disciplines={series.disciplines}
                        videoCount={series?.videos?.length || 0}
                        thumbnail={series?.videos?.length > 0 ? series?.videos[0]?.thumbnail_url : series?.coach_avatar}
                        seriesId={series?.id as string || "100"}
                        coachName={series?.coach_name}
                        coachAvatar={series?.coach_avatar}
                        difficulty={series?.difficulties?.[0]}
                        route="student"
                      />
                    </Col>
                  ))
                })}
                {/* Fill empty columns to maintain layout, only in desktop view */}
                {!isMobile && chunk.length < 4 && Array.from({ length: 4 - chunk.length }).map((_, i) => (
                  <Col key={`empty-${chunkIndex}-${i}`} style={{ height: '100%', backgroundColor: '#F1F5F9 !important' }} />
                ))}
              </Row>
            </CSSTransition>
          ))}
        </TransitionGroup>
      </>
    );
  }
  
  return (
    <Row style={{ margin: 0, padding: 0, backgroundColor: '#F1F5F9 !important' }}>
      <Col xs={isMobile ? 12 : 9} style={{ padding: '10px 10px 0 10px' }}>
        <Row style={{ marginBottom: 20 }}>
          <VideoDashboardHeader create={false}/>
        </Row>
        {
          isUserAccepted ? (
            <Row style={{ height: isMobile ? '100%': '82vh', backgroundColor: '#F1F5F9', overflow: 'auto'}}>
              {
                seriesComponent()
              }
              <SeriesPagination 
                totalCount={series?.data?.total_count || 0}
              />
            </Row>
          ): (
            <div style={{ 
              display: 'flex',
              textAlign: 'center',
              justifyContent: 'center',
              alignItems: 'center',
              height: '100%',
              padding: isMobile ? 0 : 20,
              margin: 0,
              fontSize: 32,
              fontWeight: 'bold' 
            }}>
              {
                status === 'Pending' ? (
                  <>
                    Your request is currently pending, please wait to be accepted
                  </>
                ): (
                  <>
                    You are currently not apart of any gyms, please use the search bar at the top to find a gym and request access
                  </>
                )
              }
              
            </div>
          )
        }
      </Col>
      {
        !isMobile && (
          <Col style={{ backgroundColor: 'white', height: '92vh', padding: 20}}>
            <h2>Coming Soon...</h2>
          </Col>
        )
      }
    </Row>
  )
};

export default StudentContentPage;
