import { colors } from "@/util/colors";
import { ReactNode } from "react";
import { Col, Row } from "react-bootstrap";

const IconNode = ({
  icon,
  title,
  description,
}: {
  icon: () => ReactNode;
  title: string;
  description: string;
}) => {
  return (
    <Row>
      <Row style={{ marginBottom: 10 }}>
        <Col style={{ 
          textAlign: 'center',
          display: 'flex', 
          justifyContent: 'flex-start', 
          alignItems: 'center',
          marginBottom: 10,
        }}>
          {icon()}
        </Col>
      </Row>
      <Row>
        <Col>
          <h3 style={{ color: colors.white, marginBottom: 10, }}>{title}</h3>
          <p style={{ color: colors.secondary }}>{description}</p>
        </Col>
      </Row>
    </Row>
  )
};

export default IconNode;