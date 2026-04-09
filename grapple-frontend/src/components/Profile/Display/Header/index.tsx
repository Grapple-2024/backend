import { Row, Col } from "react-bootstrap";
import { FaEdit } from "react-icons/fa";


const ProfileHeader = ({
  name, 
  description,
  setIsUpdating
}: any) => {
  return (
    <Row style={{
      padding: 20, 
      borderRadius: 10, 
      backgroundColor: 'white',
      margin: 0,
    }}>
      <Row style={{ marginBottom: 10 }}>
        <Col>
          <h2>{name}</h2>
        </Col>
        <Col style={{
          display: 'flex',
          justifyContent: 'flex-end',
        }}>
          <FaEdit 
            size={20} 
            onClick={setIsUpdating}
          />
        </Col>
      </Row>
      <Col xs={8}>
        <p>{description || ''}</p>
      </Col>
    </Row>
  )
};

export default ProfileHeader;