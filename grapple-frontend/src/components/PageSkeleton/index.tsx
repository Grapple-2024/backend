import { Col, Container, Placeholder, Row } from "react-bootstrap";
import Avatar from "../Avatar";
import Line from "../Line";

const Skeleton = () => {
  return (
    <Placeholder animation="glow">
      <div style={{ 
        boxShadow: '0 0 10px rgba(0, 0, 0, 0.1)',
        padding: 20,
        borderRadius: 10,
        marginTop: 20,
      }}>
        <Row>
          <Col xs={9} style={{ display: 'flex', justifyContent: 'flex-start' }}>
            <Placeholder xs={12} />
          </Col>
          <Col style={{ display: 'flex', justifyContent: 'flex-start' }}>
            <Placeholder xs={12} />
          </Col>
        </Row>
        <Row style={{ marginTop: 10 }}>
          <Col xs={8}>
            <Placeholder xs={12} />
          </Col>
          <Col xs={4}>
            <Placeholder xs={12} />
          </Col>
        </Row>
        <Row style={{ marginTop: 10 }}>
          <Col xs={3}>
            <Placeholder xs={12} />
          </Col>
          <Col xs={9}>
            <Placeholder xs={12} />
          </Col>
        </Row>
        <Row style={{ marginTop: 10 }}>
          <Col xs={5}>
            <Placeholder xs={12} />
          </Col>
          <Col xs={7}>
            <Placeholder xs={12} />
          </Col>
        </Row>
        <Row style={{ marginTop: 10 }}>
          <Col xs={6}>
            <Placeholder xs={12} />
          </Col>
          <Col xs={6}>
            <Placeholder xs={12} />
          </Col>
        </Row>
      </div>
    </Placeholder>
  );
}

const PageSkeleton = () => {
  return (
    <Row>
      <Col style={{ marginTop: 10 }}>
        <Container>
          <Skeleton />
          <Skeleton />
          <Skeleton />
        </Container>
      </Col>
    </Row>
  );
};

export default PageSkeleton;