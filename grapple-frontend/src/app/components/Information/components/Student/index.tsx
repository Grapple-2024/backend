import { colors } from "@/util/colors";
import { Col, Container, Row } from "react-bootstrap";
import { FaCalendarAlt, FaLightbulb, FaSearch } from "react-icons/fa";
import IconNode from "../IconNode";
import { useEffect, useState } from "react";

const Student = ({
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
              icon={() => <FaLightbulb color={colors.white} size={30} />}
              title='Learn on the go'
              description='Unlimited access to technique & training drills posted by your coach'
            />
          </Col>
          <Col>
            <IconNode 
              icon={() => <FaCalendarAlt color={colors.white} size={30} />}
              title='Stay up to date'
              description="Stay up to date with your gym's schedule and news"
            />
          </Col>
          <Col>
            <IconNode 
              icon={() => <FaSearch color={colors.white} size={30} />}
              title='Find tournaments and competitions'
              description='Find tournaments and competitions to compete in on the go'
            />
          </Col>
          <Col xs={1}></Col>
        </Row>
      </Row>
    </Container>
  );
}

export default Student;
