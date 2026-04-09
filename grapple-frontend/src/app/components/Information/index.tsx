import { useMobileContext } from "@/context/mobile";
import { useEffect, useState } from "react";
import { Col, Container, Row } from "react-bootstrap";
import {
  FaBell,
  FaLightbulb,
  FaCalendarAlt,
  FaUniversalAccess,
  FaVideo
} from "react-icons/fa";
import Coach from "./components/Coach";
import Student from "./components/Student";
import styles from './style.module.css';

const DesktopView = () => {
  const [coach, setCoach] = useState(true);
  const [coachUnderlineWidth, setCoachUnderlineWidth] = useState(0);
  const [studentUnderlineWidth, setStudentUnderlineWidth] = useState(0);

  useEffect(() => {
    if (coach) {
      setCoachUnderlineWidth(100);
      setStudentUnderlineWidth(0);
    } else {
      setCoachUnderlineWidth(0);
      setStudentUnderlineWidth(100);
    }
  }, [coach]);

  return (
    <>
      <Row className={styles.container}>
        <Col xs={4}></Col>
        <Col className={styles.coachStudentToggle} onClick={() => setCoach(true)}>
          Coaches
          <div
            className={styles.coachStudentToggleUnderline}
            style={{ width: `${coachUnderlineWidth}%` }}
          />
        </Col>
        <Col className={styles.coachStudentToggle} onClick={() => setCoach(false)}>
          Students
          <div
            className={styles.coachStudentToggleUnderline}
            style={{ width: `${studentUnderlineWidth}%` }}
          />
        </Col>
        <Col xs={4}></Col>
      </Row>
      <Container>
        {coach ? <Coach isVisible={coach} /> : <Student isVisible={!coach} />}
      </Container>
    </>
  );
};

const MobileView = () => {
  const [coach, setCoach] = useState(true);
  const [coachUnderlineWidth, setCoachUnderlineWidth] = useState(0);
  const [studentUnderlineWidth, setStudentUnderlineWidth] = useState(0);

  useEffect(() => {
    if (coach) {
      setCoachUnderlineWidth(100);
      setStudentUnderlineWidth(0);
    } else {
      setCoachUnderlineWidth(0);
      setStudentUnderlineWidth(100);
    }
  }, [coach]);

  return (
    <>
      <Row className={styles.mobileSwitcher}>
        <Col className={styles.mobileToggle} onClick={() => setCoach(true)}>
          Coaches
          <div
            className={styles.mobileToggleUnderline}
            style={{ width: `${coachUnderlineWidth}%` }}
          />
        </Col>
        <Col className={styles.mobileToggle} onClick={() => setCoach(false)}>
          Students
          <div
            className={styles.mobileToggleUnderline}
            style={{ width: `${studentUnderlineWidth}%` }}
          />
        </Col>
      </Row>
      {coach ? (
        <>
          <Row>
            <h2 className={styles.heading2}>Coaches</h2>
          </Row>
          <Row className={styles.icon}>
            <Col>
              <FaLightbulb size={100} />
            </Col>
          </Row>
          <Row>
            <Col>
              <h5 className={styles.heading5}>Connect with your students like never before & build a competitive team</h5>
            </Col>
          </Row>
          <Row className={styles.icon}>
            <Col>
              <FaVideo size={100} />
            </Col>
          </Row>
          <Row className={styles.icon}>
            <Col>
              <h5 className={styles.heading5}>Post drills and technique for your students to review outside of class</h5>
            </Col>
          </Row>
          <Row className={styles.icon}>
            <Col>
              <FaBell size={100} />
            </Col>
          </Row>
          <Row className={styles.icon}>
            <Col>
              <h5 className={styles.heading5}>Send out bulk announcements to your gym</h5>
            </Col>
          </Row>
        </>
      ) : (
        <>
          <Row>
            <h2 className={styles.heading2}>Students</h2>
          </Row>
          <Row className={styles.icon}>
            <Col>
              <FaLightbulb size={100} />
            </Col>
          </Row>
          <Row>
            <Col>
              <h5 className={styles.heading5}>Broaden your knowledge and skills outside class</h5>
            </Col>
          </Row>
          <Row className={styles.icon}>
            <Col>
              <FaUniversalAccess size={100} />
            </Col>
          </Row>
          <Row className={styles.icon}>
            <Col>
              <h5 className={styles.heading5}>Unlimited access to technique & training drills posted by your coach</h5>
            </Col>
          </Row>
          <Row className={styles.icon}>
            <Col>
              <FaCalendarAlt size={100} />
            </Col>
          </Row>
          <Row className={styles.icon}>
            <Col>
              <h5 className={styles.heading5}>Stay up to date with your gym&apos;s schedule and news</h5>
            </Col>
          </Row>
        </>
      )}
    </>
  );
};

const Information = () => {
  const { isMobile } = useMobileContext();

  return (
    <Container>
      {isMobile ? <MobileView /> : <DesktopView />}
    </Container>
  );
};

export default Information;