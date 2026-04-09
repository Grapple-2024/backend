import { Col, Row, Tab, Tabs } from "react-bootstrap";
import Create from "../Create";
import TimeCard from "../TimeCard";
import { useEffect } from "react";

const days = [
  'Sun', 
  'Mon', 
  'Tue', 
  'Wed', 
  'Thu', 
  'Fri', 
  'Sat'
];


interface SchedulerProps {
  schedule: any;
  setSchedule: (schedule: any) => void;
};

const Scheduler = ({ schedule, setSchedule }: SchedulerProps) => {
  return (
    <>
      <Tabs
        defaultActiveKey="sun"
        id="uncontrolled-tab-example"
        className="mb-3"
        fill
      >
        {days.map((day) => (
          <Tab key={day} eventKey={day.toLowerCase()} title={day}>
            <Create setSchedule={setSchedule} schedule={schedule} day={day}/>
            <div style={{
              height: '25vh',
              overflow: 'scroll',
              padding: 20,
            }}>
              {
                schedule && schedule[day.toLowerCase()]?.length > 0 ? (
                  schedule[day.toLowerCase()].sort((a: any, b: any) => {
                    const aTime = a.start.split(':').map(Number);
                    const bTime = b.start.split(':').map(Number);
                    
                    // Convert to 24-hour format
                    if (a.pm) aTime[0] += 12;
                    if (b.pm) bTime[0] += 12;
                  
                    return aTime[0] * 60 + aTime[1] - (bTime[0] * 60 + bTime[1]);
                  })?.map((times: any, i: number) => (
                    <TimeCard
                      key={i}
                      index={i}
                      title={times.title}
                      startTime={times.start}
                      endTime={times.end}
                      pm={times.pm}
                      onDelete={(index: number) => {
                        const newSchedule = schedule;
                        newSchedule[day.toLowerCase()].splice(index, 1);
                        setSchedule({ ...schedule, schedule: newSchedule });
                      }}
                    />
                  ))
                ) : (
                  <div style={{ 
                    boxShadow: '0 0 10px rgba(0, 0, 0, 0.1)',
                    padding: 20,
                    borderRadius: 10,
                    marginTop: 20
                  }}>
                    <Row>
                      <Col style={{ display: 'flex', justifyContent: 'center' }}>
                        <h3>Nothing Scheduled</h3>
                      </Col>
                    </Row>
                  </div>
                )
                
              }
            </div>
          </Tab>
        ))}
      </Tabs>
    </>
  );
};

export default Scheduler;