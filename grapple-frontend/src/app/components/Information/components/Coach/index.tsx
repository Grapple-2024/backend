import { colors } from "@/util/colors";
import { Col, Container, Row } from "react-bootstrap";
import { FaBell, FaLock, FaVideo } from "react-icons/fa";
import IconNode from "../IconNode";
import { useEffect, useState } from "react";

const Coach = ({
  isVisible
}: {
  isVisible: boolean;
}) => {
  const [triggerAnimation, setTriggerAnimation] = useState(true);

  useEffect(() => {
    if (isVisible) {
      const timer = setTimeout(() => setTriggerAnimation(false), 500); // 500ms is the duration of your transition
      return () => clearTimeout(timer); // Clean up on unmount
    }
  }, [isVisible]);

  return (
    <Container style={{
      opacity: triggerAnimation ? 0 : 1,
      transform: `translateY(${triggerAnimation ? '20px' : '0px'})`,
      transition: 'opacity 0.5s ease-in, transform 0.5s ease-in',
    }}>
      <Row style={{ height: '15vh', marginTop: 50 }}>
        <Row>
          <Col xs={1}></Col>
          <Col>
            <IconNode 
              icon={() => <FaLock color={colors.white} size={30} />}
              title='Secure Content'
              description='Complete authority of who has access to your dashboard/gym'
            />
          </Col>
          <Col>
            <IconNode 
              icon={() => <FaBell color={colors.white} size={30} />}
              title='Keep your gym informed'
              description='Notify your entire gym by sending out bulk announcements'
            />
          </Col>
          <Col>
            <IconNode 
              icon={() => <FaVideo color={colors.white} size={30} />}
              title='Engage your audience'
              description='Post drills and technique for your students to review outside of class'
            />
          </Col>
          <Col xs={1}></Col>
        </Row>
      </Row> 
    </Container>
  );
}

export default Coach;
