import { colors } from "@/util/colors";
import { useState } from "react";
import { FaEdit } from "react-icons/fa";
import { Button, Col, Container, Form, Row } from "react-bootstrap";
import { useMobileContext } from "@/context/mobile";

interface BasicInfoData {
  first_name: string;
  last_name: string;
  email: string;
  phone_number: string;
};

interface BasicInfoProps {
  user: any;
  onSave: (formData: BasicInfoData) => void;
};

const BasicInfo = ({ user, onSave }: BasicInfoProps) => {
  const { isMobile } = useMobileContext();
  
  const [isEditing, setIsEditing] = useState(false);
  const [formData, setFormData] = useState({
    first_name: user?.first_name,
    last_name: user?.last_name,
    email: user?.email,
    phone_number: user?.phone_number
  });
  
  return (
    <div style={{
      padding: 20,
      borderRadius: 10, 
      backgroundColor: 'white'
    }}>
      <Row style={{ marginBottom: 10 }}>
        <Col style={{ display: 'flex', justifyContent: 'flex-start' }}>
          <h4>Basic Info</h4>
        </Col>
        <Col style={{ display: 'flex', justifyContent: 'flex-end' }}>
          {
            !isEditing ? (
              <FaEdit 
                size={20}
                onClick={() => {
                  setIsEditing(!isEditing);
                }}
              />
            ) : null
          }
        </Col>
      </Row>
      <Row>
        <Col xs={isMobile ? 12 : 6}>
          <Form.Label htmlFor="firstNameSettings">First Name</Form.Label>
          <Form.Control
            type="text"
            id="firstNameSettings"
            aria-describedby="firstNameSettings"
            placeholder={user?.first_name}
            disabled={!isEditing}
            value={formData.first_name}
            onChange={(e) => setFormData({ ...formData, first_name: e.target.value })}
          />
        </Col>
        <Col xs={isMobile ? 12 : 6}>
          <Form.Label htmlFor="lastNameSettings">Last Name</Form.Label>
          <Form.Control
            type="text"
            id="lastNameSettings"
            aria-describedby="lastNameSettings"
            placeholder={user?.last_name}
            disabled={!isEditing}
            value={formData.last_name}
            onChange={(e) => setFormData({ ...formData, last_name: e.target.value })}
          />
        </Col>
      </Row>
      <Row style={{ marginTop: 20 }}>
        <Col xs={isMobile ? 12 : 6}>
          <Form.Label htmlFor="emailSettings">Email</Form.Label>
          <Form.Control
            type="text"
            id="emailSettings"
            aria-describedby="emailSettings"
            placeholder={user?.email}
            disabled={true}
            value={formData.email}
            onChange={(e) => setFormData({ ...formData, email: e.target.value })}
          />
        </Col>
        <Col xs={isMobile ? 12 : 6}>
          <Form.Label htmlFor="phoneNumberSettings">Phone Number</Form.Label>
          <Form.Control
            type="text"
            id="phoneNumberSettings"
            aria-describedby="phoneNumberSettings"
            placeholder={user?.phone_number}
            disabled={true}
            value={formData.phone_number}
            onChange={(e) => setFormData({ ...formData, phone_number: e.target.value })}
          />
        </Col>
      </Row>
      {
        isEditing ? (
          <Row>
            <Col style={{
              display: 'flex',
              justifyContent: 'flex-end',
            }}>
              <Button 
                variant="dark"
                onClick={() => setIsEditing(!isEditing)}
                style={{ 
                  marginTop: 20, 
                  marginRight: 10,
                  backgroundColor: 'white', 
                  borderColor: 'black', 
                  color: 'black',
                }} 
              >
                Cancel
              </Button>
              <Button style={{ 
                  marginTop: 20, 
                  backgroundColor: colors.black, 
                  borderColor: colors.black, 
                }} 
                type="submit"
                onClick={async () => {
                  setIsEditing(!isEditing);
                  await onSave(formData);
                }}
              >
                Update Basic Info
              </Button>
            </Col>
          </Row>
        ) : null
      }
    </div>
  );
};

export default BasicInfo;