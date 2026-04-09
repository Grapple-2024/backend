import { Col, Row } from "react-bootstrap";
import { FaRegTrashAlt } from "react-icons/fa";

interface TimeCardProps {
  title: string;
  index: number;
  startTime: string;
  endTime: string;
  pm: boolean;
  onDelete: (index: number) => void;
};

const TimeCard = ({
  index,
  title,
  startTime,
  endTime,
  pm,
  onDelete
}: TimeCardProps) => {
  return (
    <>
      <div style={{
        boxShadow: '0 0 10px rgba(0, 0, 0, 0.1)',
        padding: 20,
        borderRadius: 10, 
        marginTop: 20
      }}>
        <Row>
          <Col style={{ display: 'flex', justifyContent: 'flex-start' }}>
            <h3>{title}</h3>
          </Col>
          <Col style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <FaRegTrashAlt onClick={() => {
              onDelete(index);
            }}/>
          </Col>
        </Row>
        <Row style={{ marginTop: 10 }}>
          <Col>
            Start Time:{' '}{startTime}
          </Col>
          <Col>
            End Time:{' '}{endTime}
          </Col>
        </Row>
      </div>
    </>
  );
};

export default TimeCard;