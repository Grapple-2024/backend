import { useState } from "react";
import { Col, Form, InputGroup, Row } from "react-bootstrap";

const MajorFields = ({
 gym, 
 formData,
 setFormData,
}: any) => {
  return (
    <Row style={{
      padding: 20,
      margin: 0, 
      backgroundColor: 'white',
      borderRadius: 10, 
    }}>
      <Row style={{
        marginBottom: 20,
      }}>
        <Row>
          <Col>
            <h4>{
              gym?.address_line_1 + ' ' + (gym?.address_line_2 || '')
            }</h4>
          </Col>
        </Row>
        <Row>
          <Col>
            <h4>{gym?.city + ' ' + gym?.state + ', ' + gym?.zip}</h4>
          </Col>
        </Row>
      </Row>
      <Row>
        <Col xs={6}>
          <Form.Label>Public Email</Form.Label>
          <InputGroup className="mb-3">
            <Form.Control
              placeholder={gym?.public_email}
              aria-label="The public facing email"
              aria-describedby="public-email"
              disabled
            />
          </InputGroup>
        </Col>
        <Col xs={6}>
          {/* <Form.Group controlId="formFile" className="mb-3">
            <Form.Label>Banner Image</Form.Label>
            <Form.Control type="file" disabled/>
          </Form.Group> */}
        </Col>
      </Row>
      <Row>
        <Col>
          <h4>Disciplines</h4>
          <div style={{ display: 'flex', flexDirection: 'row' }}>
            {
              gym?.disciplines?.map((discipline: string, index: number) => {
                return (
                  <div key={index} style={{ marginLeft: index !== 0 ? '10px' : '0' }}>
                    <Form.Label>{discipline}</Form.Label>
                  </div>
                )
              })
            }
          </div>
        </Col>
      </Row>
    </Row>
  );
};

export default MajorFields;