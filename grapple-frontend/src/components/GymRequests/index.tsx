import { useState, useEffect } from 'react';
import { Container, Row, Col } from 'react-bootstrap';
import Table from './components/DesktopTable';
import Pagination from "@/components/Pagination";
import MobileTable from './components/MobileTable';
import DesktopTable from './components/DesktopTable';
import { useMobileContext } from '@/context/mobile';

interface GymRequestsProps {
  data: Request[];
  count: number;
  title?: string;
  kick?: boolean;
};

const ITEMS_PER_PAGE = 5;

const GymRequests = ({ data, count, title="Gym Requests", kick=false }: GymRequestsProps) => {
  const { isMobile } = useMobileContext();
  const [currentPage, setCurrentPage] = useState(1);

  
  return ( 
    <Container style={{ margin: 0, padding: 0}}>
      <Row style={{ margin: 0, padding: 0, backgroundColor: '#F1F5F9' }}>
        <Col style={{ margin: 0, padding: 0, backgroundColor: '#F1F5F9' }}>
          {data?.length > 0 ? isMobile ?(
            <MobileTable 
              defaultData={data} 
              count={count}
            />
          ) : (
            <DesktopTable 
              defaultData={data as any} 
              kick={kick}
            />
          ): (
            <div style={{
              padding: 20,
              boxShadow: '0 0 10px rgba(0, 0, 0, 0.1)',
              borderRadius: 10, 
              marginTop: 30, 
            }}>
              <h3 className="text-center">No requests at this time.</h3>
            </div>
          )}
        </Col>
      </Row>
      {/* <Row>
        {
          data?.length > 0 && (
            <Col style={{ display: 'flex', justifyContent: 'flex-end', backgroundColor: '#F1F5F9' }}>
              <Pagination count={count} onPageChange={setCurrentPage} />
            </Col>
          )
        }
      </Row> */}
    </Container>
  );
};

export default GymRequests;