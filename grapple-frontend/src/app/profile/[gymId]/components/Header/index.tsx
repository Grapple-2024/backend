import React from 'react';
import { Col, Form, Row } from "react-bootstrap";
import './styles.css';

const ProfileHeader = ({
  name,
  description,
  address_line_1,
  address_line_2,
  city,
  state,
  zip,
  disciplines,
  request = () => null
}: any) => {
  return (
    <div className="profile-header">
      <Row className="header-row">
        <Col xs={12} md={8}>
          <h2>{name}</h2>
          <p>{description || ''}</p>
        </Col>
        <Col xs={12} md={4} className="address-col">
          <h4>{address_line_1} {address_line_2 && <span>{address_line_2}</span>}</h4>
          <h4>{city} {state}, {zip}</h4>
        </Col>
      </Row>
      <Row className="disciplines-row">
        <Col xs={12}>
          <h4>Disciplines:</h4>
          <div className="disciplines-list">
            {disciplines?.map((discipline: string, index: number) => (
              <div key={index} className="discipline-item">
                <Form.Label>{discipline}</Form.Label>
              </div>
            ))}
          </div>
        </Col>
      </Row>
      <Row>
        <Col>
          {request()}
        </Col>
      </Row>
    </div>
  );
};

export default ProfileHeader;
