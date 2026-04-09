'use client';

import { Col, Placeholder, Row } from "react-bootstrap";
import { CSSTransition, TransitionGroup } from 'react-transition-group';
import './styles.css'; // Import the CSS file for animations
import { useMobileContext } from "@/context/mobile";
import VideoDashboardHeader from "@/components/VideoDashboardHeader";
import DashboardVideoCard from "@/components/DashboardVideoCard";
import SeriesPagination from "@/components/SeriesPagination";
import VideoDashboardCreateModal from "@/components/VideoDashboardCreateModal";
import { useGetSeries } from "@/hook/series";
import { useGetGym } from "@/hook/gym";

const ClientContentPage = ({
  series: initialSeries,
}: any) => {
  const { isMobile } = useMobileContext();
  const gym = useGetGym();
  const series = useGetSeries(initialSeries);
  
  const seriesComponent = () => {
    if (series?.isPending || gym?.isPending) {
      return (  
        <Placeholder animation="glow">  
          <Row style={{ 
            height: 50, 
            marginLeft: 1,
            marginRight: 1, 
          }}>
            <Placeholder />
          </Row>
          <Row style={{ 
            height: 300, 
            marginLeft: 1,
            marginRight: 1, 
          }}>
            <Placeholder />
            <Placeholder />
            <Placeholder />
          </Row>
        </Placeholder>
      );
    }
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
                        route="coach"
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
    <Row style={{ 
      margin: 0, 
      padding: 0, 
      backgroundColor: '#F1F5F9', 
      height: '100%',
      minHeight: 0,
    }}>
      <Col 
        xs={isMobile ? 12 : 9} 
        style={{ 
          padding: '10px 10px 0 10px',
          display: 'flex',          // Add flex display
          flexDirection: 'column',  // Stack children vertically
          minHeight: 0,            // Allow child content to scroll
          height: '100%'           // Take up full height
        }}
      >
        <Row style={{ marginBottom: 20, flexShrink: 0 }}>
          <VideoDashboardHeader isCoach/>
        </Row>
        <Row style={{ 
          backgroundColor: '#F1F5F9', 
          overflow: 'auto',
          flex: 1,                // Take up remaining space
          minHeight: 0           // Allow scrolling
        }}>
          {seriesComponent()}
          <SeriesPagination 
            totalCount={series?.data?.total_count || 0}
            isCoach
          />
        </Row>
      </Col>
      {!isMobile && (
        <Col style={{ 
          backgroundColor: 'white', 
          height: '100%',  // Changed from 92vh to 100%
          padding: 20
        }}>
          <h2>Coming Soon...</h2>
        </Col>
      )}
      <VideoDashboardCreateModal />
    </Row>
  )
};

export default ClientContentPage;
