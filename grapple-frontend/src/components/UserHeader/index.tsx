
import { Col, Row } from 'react-bootstrap';
import styles from './UserHeader.module.css';
import { useMobileContext } from '@/context/mobile';
import QRCode from './QrCode/QrCode';

const UserHeader = ({
  totalMembers = 0,
  gymId,
}: any) => {
  const { isMobile } = useMobileContext();

  return (
    <Row className={styles.statsContainer}>
      <h2>User Management</h2>  
      <Col xs={isMobile ? 9 : 4}>
        <Row className={styles.statsWrapper}>
          <Col className={styles.statBox}>
            <div className={styles.statLabel}>Total members</div>
            <div className={styles.statValue}>{totalMembers}</div>
          </Col>
          <Col className={styles.statBox}>
            <div className={styles.statLabel}>In-person</div>
            <div className={styles.statValue}>{totalMembers}</div>
          </Col>
          {/* <Col className={styles.statBox}>
            <div className={styles.statLabel}>Virtual</div>
            <div className={styles.statValue}>Coming Soon</div>
          </Col> */}
        </Row>
      </Col>
      <Col xs={isMobile ? 3 : 8} style={{
        display: 'flex',
        justifyContent: 'flex-start',
        alignItems: 'flex-end',
        flexDirection: 'column'
      }}>
        <QRCode defaultValue={`${process.env.NEXT_PUBLIC_APP_URL}/auth?gym_id=${gymId}`} size={250}/>
      </Col>
    </Row>
  )
};

export default UserHeader;