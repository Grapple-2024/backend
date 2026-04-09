import { useEffect, useState } from "react";
import { Col, Form, Row, Button } from "react-bootstrap";
import './styles.css';
import { FaEdit } from "react-icons/fa";
import { colors } from "@/util/colors";


const Notifications = ({
  value,
  onSave,
}: any) => {
  const [checked, setChecked] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  
  useEffect(() => {
    // Ensure value is a boolean when setting it
    setChecked(!!value);
  }, [value]); 

  return (
    <div style={{
      padding: 20,
      borderRadius: 10, 
      backgroundColor: 'white'
    }}>
      <Row style={{ marginBottom: 20 }}>
        <Col style={{ display: 'flex', justifyContent: 'flex-start' }}>
          <h4>Notifications</h4>
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
      <Form>
        <Form.Check
          type="switch"
          id="custom-switch"
          label={checked ? 'Email Notifications Enabled' : 'Email Notifications Disabled'}
          checked={checked as any}
          onChange={() => setChecked(!checked as any)}
          disabled={!isEditing}
        />
      </Form>
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
                  onSave(checked);
                  setIsEditing(!isEditing);
                }}
              >
                Save
              </Button>
            </Col>
          </Row>
        ) : null
      }
    </div>
  );
};

export default Notifications;