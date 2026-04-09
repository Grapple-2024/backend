import { useMobileContext } from '@/context/mobile';
import { GymSchedule, ScheduleItem } from '@/types/gym';
import React, { useEffect, useRef, useState } from 'react';
import { ListGroup, Container } from 'react-bootstrap';
import { FaChevronLeft, FaChevronRight } from 'react-icons/fa';

interface ProfileScheduleProps {
  days: string[];
  schedule: GymSchedule;
  selectedDay?: string;
  daily?: boolean;
}

const ProfileSchedule: React.FC<ProfileScheduleProps> = ({ days, schedule, selectedDay, daily = false }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState<number>(0);
  const [canScrollLeft, setCanScrollLeft] = useState<boolean>(false);
  const [canScrollRight, setCanScrollRight] = useState<boolean>(false);
  const { isMobile } = useMobileContext();

  useEffect(() => {
    const updateWidth = () => {
      if (containerRef.current) {
        setContainerWidth(containerRef.current.offsetWidth);
        setCanScrollLeft(containerRef.current.scrollLeft > 0);
        setCanScrollRight(
          containerRef.current.scrollWidth > containerRef.current.clientWidth &&
          containerRef.current.scrollLeft + containerRef.current.clientWidth < containerRef.current.scrollWidth
        );
      }
    };

    updateWidth();
    window.addEventListener('resize', updateWidth);

    return () => {
      window.removeEventListener('resize', updateWidth);
    };
  }, []);

  useEffect(() => {
    const handleScroll = () => {
      if (containerRef.current) {
        setCanScrollLeft(containerRef.current.scrollLeft > 0);
        setCanScrollRight(
          containerRef.current.scrollWidth > containerRef.current.clientWidth &&
          containerRef.current.scrollLeft + containerRef.current.clientWidth < containerRef.current.scrollWidth
        );
      }
    };

    if (containerRef.current) {
      containerRef.current.addEventListener('scroll', handleScroll);
    }

    return () => {
      if (containerRef.current) {
        containerRef.current.removeEventListener('scroll', handleScroll);
      }
    };
  }, []);

  const parseTime = (timeStr: string): number => {
    const [time, period] = timeStr.split(' ');
    const [hour, minute] = time.split(':').map(Number);
    return ((hour % 12 + (period.toLowerCase() === 'pm' ? 12 : 0)) * 60) + minute;
  };

  const formatTime = (timeStr: string): string => {
    let [time, period] = timeStr.split(' ');
    let [hour, minute] = time.split(':').map(Number);
    hour = hour % 12 || 12;
    return `${hour}:${minute.toString().padStart(2, '0')} ${period}`;
  };

  const mapDayToLowercase = (day: string): string => {
    return day.toLowerCase().slice(0, 3);
  };

  const groupClassesByStartTime = (classes: ScheduleItem[]): { [startTime: string]: ScheduleItem[] } => {
    return classes.reduce((acc, currentClass) => {
      const startTime = currentClass.start;
      if (!acc[startTime]) {
        acc[startTime] = [];
      }
      acc[startTime].push(currentClass);
      return acc;
    }, {} as { [startTime: string]: ScheduleItem[] });
  };

  return (
    <Container ref={containerRef} style={{ borderRadius: 10, backgroundColor: '#F1F5F9', overflow: daily ? 'hidden' : 'auto', position: 'relative', padding: 0, height: '100%' }}>
      {!daily && canScrollLeft && (
        <div style={{ position: 'absolute', left: 0, top: '50%', transform: 'translateY(-50%)', zIndex: 1, background: 'rgba(255,255,255,0.8)', padding: '10px', cursor: 'pointer' }}
          onClick={() => {
            if (containerRef.current) {
              containerRef.current.scrollBy({ left: -containerWidth / 2, behavior: 'smooth' });
            }
          }}>
          <FaChevronLeft />
        </div>
      )}

      <h4 style={{ fontSize: 24, cursor: 'pointer', fontWeight: 'bold', display: 'flex', alignItems: 'center', marginBottom: 10, }}>
          {selectedDay}
        </h4>
      <div style={{ display: daily ? 'block' : 'flex', flexDirection: 'row', whiteSpace: 'nowrap', overflowX: daily ? 'hidden' : 'auto', height: '100%' }}>
        {daily ? (
          <div style={{ padding: 0, height: '100%' }}>
            <ListGroup style={{ height: isMobile ? '' : '100%', overflowY: 'scroll', marginTop: 20 }}>
              {schedule && selectedDay && schedule[(mapDayToLowercase(selectedDay) as any)]?.length > 0 ? (
                Object.entries(groupClassesByStartTime(schedule[mapDayToLowercase(selectedDay)])).sort(([a], [b]) => parseTime(a) - parseTime(b)).map(([startTime, classes], index) => (
                  <React.Fragment key={index}>
                    <div style={{ borderBottom: '1px dotted #ddd', marginBottom: 10 }}></div>
                    <div style={{ display: 'grid', gridTemplateColumns: '90px 1fr', alignItems: 'center', marginBottom: 10 }}>
                      <div style={{ textAlign: 'right', marginRight: '10px', color: '#7A7A7A', fontWeight: 'bold' }}>
                        <p style={{ margin: 0, whiteSpace: 'nowrap' }}>{formatTime(startTime)}</p>
                      </div>
                      <div>
                        {classes.map((value, index) => (
                          <ListGroup.Item 
                            key={index}
                            style={{
                              backgroundColor: '#ffffff',
                              color: '#000',
                              padding: 5,
                              borderRadius: 8,
                              display: 'flex',
                              alignItems: 'center',
                              boxShadow: '0px 4px 6px rgba(0, 0, 0, 0.1)',
                              marginBottom: 10
                            }}
                          >
                            <div className="dot" style={{
                              width: 10,
                              height: 10,
                              backgroundColor: '#f24b4b',
                              borderRadius: '50%',
                              marginRight: 20,
                            }}></div>
                            <div style={{ flexGrow: 1 }}>
                              <p style={{ margin: 0, fontWeight: 'bold', fontSize: 14 }}>{value.title}</p>
                              <p style={{ margin: 0, color: '#7A7A7A', fontSize: 12 }}>{formatTime(value.start) + ' - ' + formatTime(value.end)}</p>
                            </div>
                          </ListGroup.Item>
                        ))}
                      </div>
                    </div>
                  </React.Fragment>
                ))
              ) : (
                <ListGroup.Item style={{
                  backgroundColor: '#ffffff',
                  color: '#000',
                  padding: 0,
                  fontSize: 14,
                  borderRadius: 8,
                  textAlign: 'center',
                  boxShadow: '0px 4px 6px rgba(0, 0, 0, 0.1)',
                }}>
                  <p style={{ padding: 10, margin: 0 }}>No Classes</p>
                </ListGroup.Item>
              )}
            </ListGroup>
          </div>
        ) : (
          days.map((day, index) => (
            <div key={index} style={{ padding: 5, display: 'inline-block', verticalAlign: 'top', minWidth: 400, maxHeight: '45vh' }}>
              <h4 style={{ marginBottom: 20, textAlign: 'center' }}>{day}</h4>
              <ListGroup>
                {schedule && schedule[mapDayToLowercase(day)]?.length > 0 ? (
                  Object.entries(groupClassesByStartTime(schedule[mapDayToLowercase(day)])).sort(([a], [b]) => parseTime(a) - parseTime(b)).map(([startTime, classes], index) => (
                    <React.Fragment key={index}>
                      <div style={{ borderBottom: '1px dotted #ddd', marginBottom: 10 }}></div>
                      <div style={{ display: 'grid', gridTemplateColumns: '90px 1fr', alignItems: 'center', marginBottom: 10 }}>
                        <div style={{ textAlign: 'right', marginRight: '10px', color: '#7A7A7A', fontWeight: 'bold' }}>
                          <p style={{ margin: 0, whiteSpace: 'nowrap' }}>{formatTime(startTime)}</p>
                        </div>
                        <div>
                          {classes.map((value: any, index: number) => (
                            <ListGroup.Item 
                              key={index}
                              style={{
                                backgroundColor: '#ffffff',
                                color: '#000',
                                padding: 15,
                                borderRadius: 8,
                                display: 'flex',
                                alignItems: 'center',
                                boxShadow: '0px 4px 6px rgba(0, 0, 0, 0.1)',
                                marginBottom: 10
                              }}
                            >
                              <div className="dot" style={{
                                width: 10,
                                height: 10,
                                backgroundColor: '#f24b4b',
                                borderRadius: '50%',
                                marginRight: 20,
                              }}></div>
                              <div style={{ flexGrow: 1 }}>
                                <p style={{ margin: 0, fontWeight: 'bold' }}>{value.title}</p>
                                <p style={{ margin: 0, color: '#7A7A7A' }}>{value.start + ' - ' + value.end}</p>
                              </div>
                            </ListGroup.Item>
                          ))}
                        </div>
                      </div>
                    </React.Fragment>
                  ))
                ) : (
                  <ListGroup.Item style={{
                    backgroundColor: '#ffffff',
                    color: '#000',
                    padding: 15,
                    borderRadius: 8,
                    textAlign: 'center',
                    boxShadow: '0px 4px 6px rgba(0, 0, 0, 0.1)',
                    marginBottom: 10
                  }}>
                    <p>No Classes</p>
                  </ListGroup.Item>
                )}
              </ListGroup>
            </div>
          ))
        )}
      </div>
      {!daily && canScrollRight && (
        <div style={{ position: 'absolute', right: 0, top: '50%', transform: 'translateY(-50%)', zIndex: 1, background: 'rgba(255,255,255,0.8)', padding: '10px', cursor: 'pointer' }}
          onClick={() => {
            if (containerRef.current) {
              containerRef.current.scrollBy({ left: containerWidth / 2, behavior: 'smooth' });
            }
          }}>
          <FaChevronRight />
        </div>
      )}
    </Container>
  );
};

export default ProfileSchedule;
